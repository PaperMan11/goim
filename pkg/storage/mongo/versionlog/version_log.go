package versionlog

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrVersionLogNotFound 表示未找到对应 DID 的版本日志
var ErrVersionLogNotFound = errors.New("version log not found")

// VersionLogModel 版本日志模型接口，可作用于 group、user 等多种实体。
// DID 语义由调用方决定（群成员同步=groupID，加入群同步=userID）。
type VersionLogModel interface {
	// IncrVersionLog 推进版本日志：原子自增版本号并追加一条变更记录。
	// did 为日志主体ID，eid 为变更实体ID，state 取 VersionStateInsert / VersionStateDelete / VersionStateUpdate。
	IncrVersionLog(ctx context.Context, did, eid string, state int32) (*model.VersionLog, error)
	// IncrVersionLogBatch 批量推进版本日志：对 N 个 EID 同时写入同一种 state，整体只推进一次 version。
	// 写入前会先清除 logs 中与这批 eid 相同的旧条目，保证每个实体只保留最新一次变更。
	IncrVersionLogBatch(ctx context.Context, did string, eids []string, state int32) (*model.VersionLog, error)
	// GetVersionLog 获取指定 DID 的版本日志。
	GetVersionLog(ctx context.Context, did string) (*model.VersionLog, error)
	// FindChangeLog 查询变更记录：根据客户端版本号查询变更记录，返回最新 limit 条。
	FindChangeLog(ctx context.Context, did string, clientVersion uint, limit int) (*model.VersionLog, error)
	// DeleteVersionLog 删除指定 DID 的版本日志。
	DeleteVersionLog(ctx context.Context, did string) error
}

type defaultVersionLogModel struct {
	versionMod *mon.Model
}

func NewVersionLogModel(versionMod *mon.Model) VersionLogModel {
	return &defaultVersionLogModel{versionMod: versionMod}
}

func (m *defaultVersionLogModel) IncrVersionLog(ctx context.Context, did, eid string, state int32) (*model.VersionLog, error) {
	return m.IncrVersionLogBatch(ctx, did, []string{eid}, state)
}

// IncrVersionLogBatch 批量推进版本日志：使用单条聚合 Pipeline 完成 自增版本 + 去重旧条目 + 追加新条目。
// 整体是一次 FindOneAndUpdate 原子命令，比两步（自增→push）更一致，也避免了批量场景产生 N 个不同 version。
func (m *defaultVersionLogModel) IncrVersionLogBatch(ctx context.Context, did string, eids []string, state int32) (*model.VersionLog, error) {
	now := timex.Now()
	if len(eids) == 0 {
		return nil, errors.New("eids must not be empty")
	}

	// 新日志元素模板：version 用 "$version"（聚合表达式），引用 stage 2 自增后的字段值。
	newElems := make([]bson.M, 0, len(eids))
	for _, eid := range eids {
		newElems = append(newElems, bson.M{
			"e_id":        eid,
			"version":     "$version",
			"state":       state,
			"last_update": now,
		})
	}

	pipeline := mongo.Pipeline{
		// Stage 1: 把 eids 暂存为临时字段 delete_e_ids，供 stage 3 $filter 使用
		{{Key: "$addFields", Value: bson.M{
			"delete_e_ids": eids,
		}}},
		// Stage 2: 对 upsert 文档补零值，对已有文档保持 version/deleted 原样
		{{Key: "$set", Value: bson.M{
			"d_id":    bson.M{"$ifNull": bson.A{"$d_id", did}},
			"version": bson.M{"$ifNull": bson.A{"$version", model.FirstVersion}},
			"deleted": bson.M{"$ifNull": bson.A{"$deleted", model.DefaultDeleteVersion}},
			"logs":    bson.M{"$ifNull": bson.A{"$logs", []model.VersionLogElem{}}},
		}}},
		// Stage 3: version 自增 + last_update 更新
		{{Key: "$set", Value: bson.M{
			"version":     bson.M{"$add": bson.A{"$version", 1}},
			"last_update": now,
		}}},
		// Stage 4: 剔除 logs 中与本次 eids 相同的旧条目（每个实体只保留最新一条）
		//
		// 示例：DB 当前 logs 为 [user_A:insert@v10, user_B:delete@v15, user_A:delete@v30]
		// 本次调用 IncrVersionLogBatch(ctx, "group_1", ["user_A","user_C"], Insert)
		//
		//   $filter 遍历旧 logs，对每条 log 判断：log.e_id ∈ delete_e_ids ?
		//     user_A (insert@v10) → 是 → 剔除
		//     user_B (delete@v15) → 否 → 保留
		//     user_A (delete@v30) → 是 → 剔除
		//
		//   Stage 4 之后 logs 变为 [user_B:delete@v15]
		//   Stage 5 $concatArrays 追加 [user_A:insert@v43, user_C:insert@v43]
		//   最终 logs = [user_B:delete@v15, user_A:insert@v43, user_C:insert@v43]
		//   旧的 user_A 历史被清除，每实体只保留最新状态
		{{Key: "$set", Value: bson.M{
			"logs": bson.M{
				"$filter": bson.M{
					"input": "$logs",
					"as":    "log",
					"cond": bson.M{
						"$not": bson.M{"$in": bson.A{"$$log.e_id", "$delete_e_ids"}},
					},
				},
			},
		}}},
		// Stage 5: 追加新元素（version 取 stage 3 计算后的 "$version"）
		{{Key: "$set", Value: bson.M{
			"logs": bson.M{"$concatArrays": bson.A{"$logs", newElems}},
		}}},
		// Stage 6: 移除临时字段
		{{Key: "$unset", Value: "delete_e_ids"}},
		// // Stage 7: 判断是否需要裁剪 logs
		// {{Key: "$set", Value: bson.M{
		// 	"_need_trim": bson.M{"$gt": bson.A{bson.M{"$size": "$logs"}, MaxLogEntries}},
		// }}},
		// // Stage 8: 裁剪 logs 保留最新 MaxLogEntries 条
		// {{Key: "$set", Value: bson.M{
		// 	"logs": bson.M{"$cond": bson.M{
		// 		"if":   "$_need_trim",
		// 		"then": bson.M{"$slice": bson.A{"$logs", -MaxLogEntries}},
		// 		"else": "$logs",
		// 	}},
		// }}},
		// // Stage 9: 推进 deleted 水位线（裁剪后首条日志的 version 即为新的清理边界）
		// {{Key: "$set", Value: bson.M{
		// 	"deleted": bson.M{"$cond": bson.M{
		// 		"if": "$_need_trim",
		// 		"then": bson.M{"$getField": bson.M{
		// 			"field": "version",
		// 			"input": bson.M{"$arrayElemAt": bson.A{"$logs", 0}},
		// 		}},
		// 		"else": "$deleted",
		// 	}},
		// }}},
		// // Stage 10: 移除临时字段
		// {{Key: "$unset", Value: "_need_trim"}},
	}

	// 投影去掉 logs，避免把整个大数组拉回客户端
	opt := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After).
		SetProjection(bson.M{"logs": 0})

	result, err := m.versionMod.Collection.FindOneAndUpdate(ctx, bson.M{"d_id": did}, pipeline, opt)
	if err != nil {
		return nil, err
	}
	var table model.VersionLogTable
	if err := result.Decode(&table); err != nil {
		return nil, err
	}

	// 在客户端内存里拼装"本次新增的 N 条"elem（DB 里 projection 已经把 logs 投影掉了）
	table.Logs = make([]model.VersionLogElem, 0, len(eids))
	for _, eid := range eids {
		table.Logs = append(table.Logs, model.VersionLogElem{
			EID:        eid,
			State:      state,
			Version:    table.Version,
			LastUpdate: table.LastUpdate,
		})
	}
	return table.VersionLog(), nil
}

func (m *defaultVersionLogModel) GetVersionLog(ctx context.Context, did string) (*model.VersionLog, error) {
	var table model.VersionLogTable
	result, err := m.versionMod.Collection.FindOne(ctx, bson.M{"d_id": did})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// 文档不存在 → 自动初始化
			return m.initVersionLog(ctx, did)
		}
		return nil, err
	}
	if err := result.Decode(&table); err != nil {
		return nil, err
	}
	if table.Version == 0 {
		table.Version = model.FirstVersion
	}
	// 如果 LastUpdate 是零值，用当前时间兜底
	if table.LastUpdate.IsZero() && len(table.Logs) > 0 {
		// 取最后一条日志的时间
		table.LastUpdate = table.Logs[len(table.Logs)-1].LastUpdate
	}
	return table.VersionLog(), nil
}

func (m *defaultVersionLogModel) initVersionLog(ctx context.Context, did string) (*model.VersionLog, error) {
	now := time.Now()
	filter := bson.M{"d_id": did}
	update := bson.M{"$setOnInsert": bson.M{
		"d_id":        did,
		"version":     model.FirstVersion,
		"deleted":     model.DefaultDeleteVersion,
		"last_update": now,
		"logs":        []model.VersionLogElem{},
	}}
	opt := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var table model.VersionLogTable
	result, err := m.versionMod.Collection.FindOneAndUpdate(ctx, filter, update, opt)
	if err != nil {
		return nil, err
	}
	if err := result.Decode(&table); err != nil {
		return nil, err
	}
	return table.VersionLog(), nil
}

// FindChangeLog 查询指定 DID 在客户端版本之后的变更记录。
//
// 语义（全有或全无策略）：
//   - 仅返回 logs 中 version > clientVersion 的条目，按 version 升序（数组原始顺序）
//   - limit > 0 时：若变更数 ≤ limit 则全部返回；若变更数 > limit 则返回空数组，
//     暗示客户端"落后太多，应走全量同步"
//   - limit <= 0 时：返回全部过滤结果，不做条数限制
//   - 兼容性校验：若 clientVersion ≥ DB 的 deleted 水位线（说明该版本之前的日志已被清理），
//     或 clientVersion > DB 当前 version，返回空数组让客户端走全量
//   - 文档不存在返回 ErrVersionLogNotFound
//   - 返回值里的 Version/Deleted/LastUpdate 始终是 DB 最新值
//
// fast path：clientVersion==0 && limit==0 时直接走 FindOne，跳过聚合管道。
//
// 示例：DB logs = [v11:insert@A, v15:update@B, v20:delete@C, v50:update@A]
//
//	FindChangeLog(ctx, "group_1", clientVersion=10, limit=2)
//	→ 变更数=4 > limit=2 → 返回 Logs=[]（空数组），客户端应走全量同步
//
//	FindChangeLog(ctx, "group_1", clientVersion=10, limit=10)
//	→ 变更数=4 ≤ limit=10 → 返回 Logs=[v11, v15, v20, v50]，Version=50
func (m *defaultVersionLogModel) FindChangeLog(ctx context.Context, did string, clientVersion uint, limit int) (*model.VersionLog, error) {
	// fast path：首次全量同步，直接 FindOne
	if clientVersion == 0 && limit == 0 {
		return m.GetVersionLog(ctx, did)
	}

	// Stage 1: $match 定位文档
	// Stage 2: 兼容性校验：clientVersion 过大/过小 → 清空 logs（走全量）
	// Stage 3: 过滤出 version > clientVersion 的条目
	// Stage 4: 计算过滤后条数
	// Stage 5: 全有或全无判断（仅当 limit > 0 时生效）
	pipeline := []bson.M{
		{"$match": bson.M{"d_id": did}},
		{"$addFields": bson.M{
			// 兼容性校验：客户端版本落在已清理范围 / 超过 DB 当前版本 → 清空 logs
			"logs": bson.M{"$cond": bson.M{
				"if": bson.M{"$or": bson.A{
					bson.M{"$gte": bson.A{"$deleted", clientVersion}}, // clientVersion 在已清理范围
					bson.M{"$gt": bson.A{clientVersion, "$version"}},  // clientVersion > DB 当前版本
				}},
				"then": []model.VersionLogElem{},
				"else": "$logs",
			}},
		}},
		{"$addFields": bson.M{
			"logs": bson.M{"$filter": bson.M{
				"input": "$logs",
				"as":    "l",
				"cond":  bson.M{"$gt": bson.A{"$$l.version", clientVersion}},
			}},
		}},
		{"$addFields": bson.M{
			"log_len": bson.M{"$size": "$logs"},
		}},
	}

	// limit > 0 时：超限则清空 logs（空数组 = 走全量同步）
	if limit > 0 {
		pipeline = append(pipeline, bson.M{
			"$addFields": bson.M{
				"logs": bson.M{"$cond": bson.M{
					"if":   bson.M{"$gt": bson.A{"$log_len", limit}},
					"then": []model.VersionLogElem{},
					"else": "$logs",
				}},
			},
		})
	}

	// 最终投影
	pipeline = append(pipeline, bson.M{
		"$project": bson.M{
			"_id":         0,
			"d_id":        1,
			"version":     1,
			"deleted":     1,
			"last_update": 1,
			"logs":        1,
		},
	})

	cur, err := m.versionMod.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	if !cur.Next(ctx) {
		// 文档不存在 → 自动初始化并返回空 Logs（调用方应走全量同步）
		return m.initVersionLog(ctx, did)
	}
	var table model.VersionLogTable
	if err := cur.Decode(&table); err != nil {
		return nil, err
	}

	return table.VersionLog(), nil
}

// DeleteVersionLog 删除指定 DID 的版本日志文档。
// 常用于群解散/用户退出等场景，删除后下次同步会自动初始化空文档。
func (m *defaultVersionLogModel) DeleteVersionLog(ctx context.Context, did string) error {
	_, err := m.versionMod.Collection.DeleteOne(ctx, bson.M{"d_id": did})
	return err
}
