package msg

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/convert"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrMsgNotFound = errors.New("message not found")
	ErrInvalidSeq  = errors.New("msg seq must be positive")
)

type MsgModel interface {
	// Insert 向分桶文档中插入一条消息（Upsert 文档 + $set 槽位）
	Insert(ctx context.Context, conversationID string, msg *model.MsgDataModel) error
	// BatchInsert 批量插入消息到分桶文档
	BatchInsert(ctx context.Context, conversationID string, msgs []*model.MsgDataModel) error
	// FindBySeq 按 seq 查询单条消息
	FindBySeq(ctx context.Context, conversationID string, seq int64) (*model.MsgDataModel, error)
	// FindInfoBySeq 按 seq 查询单条消息的完整槽位（含 Revoke/DelList/IsRead），用于引用消息回源判断撤回状态
	FindInfoBySeq(ctx context.Context, conversationID string, seq int64) (*model.MsgInfoModel, error)
	// FindBySeqs 按 seq 列表查询消息
	FindBySeqs(ctx context.Context, conversationID string, seqs []int64) (map[string][]*model.MsgDataModel, error)
	// FindByConversationID 按 seq 范围查询消息（返回完整槽位，含 Revoke/DelList/IsRead）
	FindByConversationID(ctx context.Context, conversationID string, seqStart int64, seqEnd int64, limit int) ([]*model.MsgInfoModel, error)
	// FindLatestMsg 查询会话最新消息
	FindLatestMsg(ctx context.Context, conversationID string) (*model.MsgDataModel, error)
	// GetMaxSeq 获取会话最大 seq（从最后一个文档的最后一个非空槽位获取）
	GetMaxSeq(ctx context.Context, conversationID string) (int64, error)
	// GetMinSeq 获取会话最小 seq（从第一个文档的第一个非空槽位获取）
	GetMinSeq(ctx context.Context, conversationID string) (int64, error)
	// UpdateRevoke 更新消息撤回状态（设置 msgs.N.revoke，不修改原始 Msg）
	UpdateRevoke(ctx context.Context, conversationID string, seq int64, revoke *model.RevokeModel) error
	// DeleteBySeq 物理删除消息（$unset 清空槽位）
	DeleteBySeq(ctx context.Context, conversationID string, seqs []int64) error
	// DeleteByTimestamp 按时间戳物理删除过期文档，返回 conversationID -> 被删除消息的 seq 列表
	DeleteByTimestamp(ctx context.Context, conversationIDs []string, timestamp int64) (map[string][]int64, error)
	// 标记删除（将 userID 加入目标槽位的 del_list）
	MarkDeleteBySeqs(ctx context.Context, userID string, conversationID string, seqs []int64) error
	// 标记消息已读
	MarkReadBySeqRange(ctx context.Context, userID string, conversationID string, seqStart, seqEnd int64) error
	MarkReadBySeqs(ctx context.Context, userID string, conversationID string, seqs []int64) error
	// GetLastMessageSeqByTimestamp 获取会话最新消息 seq（按时间戳）
	GetLastMessageSeqByTimestamp(ctx context.Context, conversationID string, timestamp int64) (int64, error)
	// SearchMessage
	SearchMessage(ctx context.Context, req *model.SearchMessageReq) ([]*model.MsgInfoModel, error)
}

type defaultMsgModel struct {
	mod *mon.Model
}

func NewMsgModel(mod *mon.Model, barrier syncx.SingleFlight, rdb goredis.UniversalClient, enableCache bool) MsgModel {
	m := &defaultMsgModel{
		mod: mod,
	}
	if !enableCache || rdb == nil {
		return m
	}
	return NewCachedMsgModel(m, rdb, barrier)
}

// docIDPrefix 返回 conversationID 对应的一批 DocID 的前缀匹配条件。
// DocID 格式为 "conversationID:seqSuffix"，同一会话的所有文档都以此为前缀。
// 用 [$gte, $lt) 半开区间代替 $regex：conversationID 可能含正则元字符（如 "."、"+"、"("），
// 直接拼进正则会造成误匹配甚至 invalid regex 报错；半开区间按字节序比较，语义等价且更安全。
// 上界取 ";"（':' 的下一个 ASCII 字符）：seqSuffix 以数字 0-9 开头，均落在 "conv:" 与 "conv:;" 之间。
func docIDPrefix(conversationID string) bson.M {
	prefix := conversationID + ":"
	return bson.M{"doc_id": bson.M{"$gte": prefix, "$lt": prefix + ";"}}
}

// buildSlotUpdate 构建按索引位置精确"追加/写入"槽位的聚合管道 $set 片段。
// 思路：用 $let + $range 从 0..targetIndex 生成一个长度为 targetIndex+1 的数组，
// 对每个位置 i：i==targetIndex → 写目标 msgInfo；i<targetIndex 且文档已有该槽位 → 保留原值；否则填 nil（稀疏占位）。
// 这样无论文档是否存在、是否中间有空槽，一次 Update 就能把目标槽位"追加"到位，不会因为先写 msgs.50 导致文档重建时产生 0..49 的膨胀。
func buildSlotUpdate(targetIndex int64, msgInfo any) bson.M {
	iExpr := "$$i"
	newSize := bson.M{"$max": []any{bson.M{"$size": bson.M{"$ifNull": []any{"$msgs", []any{}}}}, targetIndex + 1}}
	return bson.M{
		"$let": bson.M{
			"vars": bson.M{
				"old": bson.M{"$ifNull": []any{"$msgs", []any{}}},
			},
			"in": bson.M{
				"$map": bson.M{
					"input": bson.M{"$range": []any{0, newSize, 1}},
					"as":    "i",
					"in": bson.M{
						"$cond": bson.M{
							"if":   bson.M{"$eq": []any{iExpr, targetIndex}},
							"then": msgInfo,
							"else": bson.M{
								"$cond": bson.M{
									"if":   bson.M{"$lt": []any{iExpr, bson.M{"$size": "$$old"}}},
									"then": bson.M{"$arrayElemAt": []any{"$$old", iExpr}},
									"else": nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildMultiSlotUpdate 构建同一文档下多个槽位的聚合管道 $set 片段（用于 BatchInsert）。
// 输入 slots 是按 msgIndex -> MsgInfo 去重后的映射。整体策略与 buildSlotUpdate 相同，
// 只是 $cond 分支内依次判断「当前 i 是否命中 slots 中的任一目标索引」，从而一次管道写入多个槽位。
func buildMultiSlotUpdate(slots map[int64]*model.MsgInfoModel) bson.M {
	maxIdx := int64(-1)
	for idx := range slots {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	if maxIdx < 0 {
		return bson.M{"$ifNull": []any{"$msgs", []any{}}}
	}

	// 从最内层开始，构建形如：if i==maxIdx -> s[maxIdx] else (if i==maxIdx-1 -> s[maxIdx-1] else ...)
	// 最后套 default 分支：i<len(old) 取原值，否则 nil（$$old 在下面的 $let.vars 里被绑定）
	var elseExpr = bson.M{
		"$cond": bson.M{
			"if":   bson.M{"$lt": []any{"$$i", bson.M{"$size": "$$old"}}},
			"then": bson.M{"$arrayElemAt": []any{"$$old", "$$i"}},
			"else": nil,
		},
	}

	// 按索引升序构建 switch（最内层→最外层），为了可读性先收集升序切片
	idxs := make([]int64, 0, len(slots))
	for idx := range slots {
		idxs = append(idxs, idx)
	}
	// 从大到小嵌套，大索引的判断在外层
	slices.Sort(idxs)
	for i := len(idxs) - 1; i >= 0; i-- {
		idx := idxs[i]
		elseExpr = bson.M{
			"$cond": bson.M{
				"if":   bson.M{"$eq": []any{"$$i", idx}},
				"then": slots[idx],
				"else": elseExpr,
			},
		}
	}

	newSize := bson.M{"$max": []any{bson.M{"$size": "$$old"}, maxIdx + 1}}
	return bson.M{
		"$let": bson.M{
			"vars": bson.M{
				"old": bson.M{"$ifNull": []any{"$msgs", []any{}}},
			},
			"in": bson.M{
				"$map": bson.M{
					"input": bson.M{"$range": []any{0, newSize, 1}},
					"as":    "i",
					"in":    elseExpr,
				},
			},
		},
	}
}

// Insert 向分桶文档中插入一条消息（支持追加写入）。
// 采用聚合管道 Update + Upsert：无论文档是否存在、msgIndex 是否大于当前数组长度，
// 都能把目标槽位精确写入到正确索引位置，中间位置保留原值或填 nil（避免稀疏膨胀）。
// 等价于"把这条消息追加到该桶文档的 seq%100 位置"语义。
func (m *defaultMsgModel) Insert(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
	doc := &model.MsgDocModel{}
	docID := doc.GetDocID(conversationID, msg.Seq)
	msgIndex := doc.GetMsgIndex(msg.Seq)

	msgInfo := &model.MsgInfoModel{Msg: msg}

	filter := bson.M{"doc_id": docID}
	update := bson.A{
		bson.M{
			"$set": bson.M{
				"doc_id": docID, // upsert 时写入，已存在则不变
				"msgs":   buildSlotUpdate(msgIndex, msgInfo),
			},
		},
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := m.mod.UpdateOne(ctx, filter, update, opts)
	return err
}

// BatchInsert 批量插入消息（按 docID 分桶后追加写入，一个文档一次 UpdateOne）。
// 1) 把 msgs 按 docID 分组，每组再按 msgIndex 去重（同一 seq 覆盖写）。
// 2) 对每个 docID 执行一次聚合管道 Update，把所有槽位一次性追加到 msgs 对应索引位置。
// 效果：N 条消息若落在 M 个桶文档，只执行 M 次 UpdateOne（M <= N），远优于旧实现的 N 次。
func (m *defaultMsgModel) BatchInsert(ctx context.Context, conversationID string, msgs []*model.MsgDataModel) error {
	if len(msgs) == 0 {
		return nil
	}
	docHelper := &model.MsgDocModel{}

	// docID -> map[msgIndex]MsgInfoModel（同桶同索引下后来者覆盖前者，seq 全局唯一时不会出现冲突）
	grouped := make(map[string]map[int64]*model.MsgInfoModel)
	for _, msg := range msgs {
		if msg.Seq < 0 {
			return fmt.Errorf("conversationID %s, clientMsgID %s: %w, got %d", conversationID, msg.ClientMsgID, ErrInvalidSeq, msg.Seq)
		}
		docID := docHelper.GetDocID(conversationID, msg.Seq)
		msgIndex := docHelper.GetMsgIndex(msg.Seq)
		if _, ok := grouped[docID]; !ok {
			grouped[docID] = make(map[int64]*model.MsgInfoModel)
		}
		grouped[docID][msgIndex] = &model.MsgInfoModel{Msg: msg}
	}

	for docID, slots := range grouped {
		filter := bson.M{"doc_id": docID}
		update := bson.A{
			bson.M{
				"$set": bson.M{
					"doc_id": docID,
					"msgs":   buildMultiSlotUpdate(slots),
				},
			},
		}
		opts := options.UpdateOne().SetUpsert(true)
		if _, err := m.mod.UpdateOne(ctx, filter, update, opts); err != nil {
			return err
		}
	}
	return nil
}

// FindBySeq 按 seq 查询单条消息：定位文档 → 从 msgs 数组取指定索引槽位
func (m *defaultMsgModel) FindBySeq(ctx context.Context, conversationID string, seq int64) (*model.MsgDataModel, error) {
	doc := &model.MsgDocModel{}
	docID := doc.GetDocID(conversationID, seq)
	msgIndex := doc.GetMsgIndex(seq)

	var result model.MsgDocModel
	err := m.mod.FindOne(ctx, &result, bson.M{"doc_id": docID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrMsgNotFound
		}
		return nil, err
	}

	if int(msgIndex) < len(result.Msgs) && result.Msgs[msgIndex] != nil && result.Msgs[msgIndex].Msg != nil {
		return result.Msgs[msgIndex].Msg, nil
	}
	return nil, ErrMsgNotFound
}

// FindInfoBySeq 按 seq 查询单条消息的完整槽位（含 Revoke/DelList/IsRead），用于引用消息回源判断撤回状态
func (m *defaultMsgModel) FindInfoBySeq(ctx context.Context, conversationID string, seq int64) (*model.MsgInfoModel, error) {
	doc := &model.MsgDocModel{}
	docID := doc.GetDocID(conversationID, seq)
	msgIndex := doc.GetMsgIndex(seq)

	var result model.MsgDocModel
	err := m.mod.FindOne(ctx, &result, bson.M{"doc_id": docID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrMsgNotFound
		}
		return nil, err
	}
	if int(msgIndex) < len(result.Msgs) && result.Msgs[msgIndex] != nil {
		return result.Msgs[msgIndex], nil
	}
	return nil, ErrMsgNotFound
}

// FindBySeqs 按 seq 列表查询消息：定位文档 → 从 msgs 数组取指定索引槽位
func (m *defaultMsgModel) FindBySeqs(ctx context.Context, conversationID string, seqs []int64) (map[string][]*model.MsgDataModel, error) {
	doc := &model.MsgDocModel{}

	docs := make(map[string][]int64) // docID -> seqs index
	for _, seq := range seqs {
		docID := doc.GetDocID(conversationID, seq)
		docs[docID] = append(docs[docID], doc.GetMsgIndex(seq))
	}

	result := make(map[string][]*model.MsgDataModel)
	for docID, indexes := range docs {
		pipeline := []bson.M{
			{"$match": bson.M{"doc_id": docID}},
			{"$project": bson.M{
				"_id":    0,
				"doc_id": 1,
				"msgs": bson.M{
					"$map": bson.M{
						"input": indexes,
						"as":    "idx",
						"in":    bson.M{"$arrayElemAt": []interface{}{"$msgs", "$$idx"}},
					},
				},
			}},
		}
		if err := m.mod.Aggregate(ctx, &doc, pipeline); err != nil {
			return nil, err
		}
		msgs := make([]*model.MsgDataModel, 0, len(doc.Msgs))
		for _, info := range doc.Msgs {
			if info != nil && info.Msg != nil {
				msgs = append(msgs, info.Msg)
			}
		}
		result[docID] = msgs
	}
	return result, nil
}

// FindByConversationID 按 seq 范围查询消息：定位文档范围 → 遍历提取（返回完整槽位，含 Revoke/DelList/IsRead）
func (m *defaultMsgModel) FindByConversationID(ctx context.Context, conversationID string, seqStart, seqEnd int64, limit int) ([]*model.MsgInfoModel, error) {
	doc := &model.MsgDocModel{}

	// seqStart 和 seqEnd 都为 0 时查询全部
	var filter bson.M
	if seqStart > 0 || seqEnd > 0 {
		startDocID := doc.GetDocID(conversationID, seqStart)
		endDocID := doc.GetDocID(conversationID, seqEnd)
		filter = bson.M{"doc_id": bson.M{"$gte": startDocID, "$lte": endDocID}}
	} else {
		filter = docIDPrefix(conversationID)
	}

	opts := options.Find().SetSort(bson.M{"doc_id": 1})

	var docs []*model.MsgDocModel
	err := m.mod.Find(ctx, &docs, filter, opts)
	if err != nil {
		return nil, err
	}

	var result []*model.MsgInfoModel
	for _, d := range docs {
		for _, info := range d.Msgs {
			if info != nil && info.Msg != nil {
				if (seqStart > 0 || seqEnd > 0) && (info.Msg.Seq < seqStart || info.Msg.Seq > seqEnd) {
					continue
				}
				result = append(result, info)
				if limit > 0 && len(result) >= limit {
					return result, nil
				}
			}
		}
	}
	return result, nil
}

// FindLatestMsg 查询会话最新消息：找最后一个文档 → 从后往前找第一个非空槽位
func (m *defaultMsgModel) FindLatestMsg(ctx context.Context, conversationID string) (*model.MsgDataModel, error) {
	// var doc model.MsgDocModel
	// opts := options.FindOne().SetSort(bson.M{"doc_id": -1})
	// err := m.mod.FindOne(ctx, &doc, docIDPrefix(conversationID), opts)
	// if err != nil {
	// 	if err == mongo.ErrNoDocuments {
	// 		return nil, ErrMsgNotFound
	// 	}
	// 	return nil, err
	// }

	// for i := len(doc.Msgs) - 1; i >= 0; i-- {
	// 	if doc.Msgs[i] != nil && doc.Msgs[i].Msg != nil && doc.Msgs[i].Msg.Status < constant.MsgStatusHasDeleted {
	// 		return doc.Msgs[i].Msg, nil
	// 	}
	// }
	// return nil, ErrMsgNotFound

	pipeline := bson.A{
		bson.E{"$match", docIDPrefix(conversationID)},
		bson.E{"$match", bson.M{"msgs.msg.status": bson.M{"$lt": constant.MsgStatusHasDeleted}}},
		bson.E{"$sort", bson.M{"doc_id": -1}},
		bson.E{"$limit", 1},
		bson.E{"$project", bson.M{"_id": 0, "doc_id": 0, "msgs": 1}},
		bson.E{"$unwind", "$msgs"},
		bson.E{"$match", bson.M{"msgs.msg.status": bson.M{"$lt": constant.MsgStatusHasDeleted}}},
		bson.E{"$sort", bson.M{"msgs.msg.seq": -1}},
		bson.E{"$limit", 1},
	}
	var result model.MsgInfoModel
	err := m.mod.Aggregate(ctx, &result, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrMsgNotFound
		}
		return nil, err
	}
	if result.Msg != nil {
		return result.Msg, nil
	}
	return nil, ErrMsgNotFound
}

// GetMaxSeq 获取会话最大 seq：最后一个文档的最后一个非空槽位的 seq
func (m *defaultMsgModel) GetMaxSeq(ctx context.Context, conversationID string) (int64, error) {
	var doc model.MsgDocModel
	opts := options.FindOne().SetSort(bson.M{"doc_id": -1})
	err := m.mod.FindOne(ctx, &doc, docIDPrefix(conversationID), opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}

	for i := len(doc.Msgs) - 1; i >= 0; i-- {
		if doc.Msgs[i] != nil && doc.Msgs[i].Msg != nil {
			return doc.Msgs[i].Msg.Seq, nil
		}
	}
	return 0, nil
}

// GetMinSeq 获取会话最小 seq：第一个文档的第一个非空槽位的 seq
func (m *defaultMsgModel) GetMinSeq(ctx context.Context, conversationID string) (int64, error) {
	var doc model.MsgDocModel
	opts := options.FindOne().SetSort(bson.M{"doc_id": 1})
	err := m.mod.FindOne(ctx, &doc, docIDPrefix(conversationID), opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}

	for _, info := range doc.Msgs {
		if info != nil && info.Msg != nil {
			return info.Msg.Seq, nil
		}
	}
	return 0, nil
}

// UpdateRevoke 更新消息撤回状态：定位文档 → $set msgs.N.revoke（不修改原始 Msg）
func (m *defaultMsgModel) UpdateRevoke(ctx context.Context, conversationID string, seq int64, revoke *model.RevokeModel) error {
	doc := &model.MsgDocModel{}
	docID := doc.GetDocID(conversationID, seq)
	msgIndex := doc.GetMsgIndex(seq)

	filter := bson.M{"doc_id": docID}
	update := bson.M{
		"$set": bson.M{
			"msgs." + convert.ToString(msgIndex) + ".revoke": revoke,
		},
	}
	_, err := m.mod.UpdateOne(ctx, filter, update)
	return err
}

// DeleteBySeq 物理删除消息：按 docID 分组，每个文档一次 $unset 清空其命中的所有槽位。
// 同文档下的多个 seq 合并成一次 UpdateOne（旧实现是逐 seq 一次往返），减少 N 倍网络往返。
func (m *defaultMsgModel) DeleteBySeq(ctx context.Context, conversationID string, seqs []int64) error {
	if len(seqs) == 0 {
		return nil
	}

	doc := &model.MsgDocModel{}
	docIdxMap := make(map[string][]int64)
	for _, seq := range seqs {
		docID := doc.GetDocID(conversationID, seq)
		msgIndex := doc.GetMsgIndex(seq)
		docIdxMap[docID] = append(docIdxMap[docID], msgIndex)
	}

	for docID, indexes := range docIdxMap {
		unsetFields := make(bson.M, len(indexes))
		for _, msgIndex := range indexes {
			unsetFields["msgs."+convert.ToString(msgIndex)] = ""
		}
		update := bson.M{"$unset": unsetFields}
		if _, err := m.mod.UpdateOne(ctx, bson.M{"doc_id": docID}, update); err != nil {
			return err
		}
	}
	return nil
}

// DeleteByTimestamp 按时间戳物理删除过期文档（整文档删除），返回 conversationID -> 被删除消息的 seq 列表。
// timestamp 与 MsgDataModel.SendTime 同为毫秒时间戳；只有"文档内全部有效消息都已过期"的文档才整体删除，
// 只要存在任一条晚于 timestamp 的消息就保留整个文档（不做文档内部分槽位清理）。
// 返回被删 seq 是为了让缓存层精确失效这些 seq 的缓存键，保持缓存与 DB 一致。
func (m *defaultMsgModel) DeleteByTimestamp(ctx context.Context, conversationIDs []string, timestamp int64) (map[string][]int64, error) {
	deleted := make(map[string][]int64)
	if len(conversationIDs) == 0 {
		return deleted, nil
	}

	for _, conversationID := range conversationIDs {
		// 投影只取判定所需的 doc_id 与各槽位的 seq/send_time，避免把整份消息正文加载进内存。
		opts := options.Find().SetProjection(bson.M{
			"doc_id":             1,
			"msgs.msg.seq":       1,
			"msgs.msg.send_time": 1,
		})
		var docs []*model.MsgDocModel
		if err := m.mod.Find(ctx, &docs, docIDPrefix(conversationID), opts); err != nil {
			return nil, err
		}

		var expiredDocIDs []string
		var expiredSeqs []int64
		for _, d := range docs {
			var docSeqs []int64
			// 默认视为"可整体回收"：没有任何有效消息的空文档（历史物理删除留下的空壳）同样应当回收，
			// 而不是像旧实现那样因 hasAnyMsg=false 被永久跳过。
			allExpired := true
			for _, info := range d.Msgs {
				if info == nil || info.Msg == nil {
					continue
				}
				if info.Msg.SendTime > timestamp {
					allExpired = false
					break
				}
				docSeqs = append(docSeqs, info.Msg.Seq)
			}
			if allExpired {
				expiredDocIDs = append(expiredDocIDs, d.DocID)
				expiredSeqs = append(expiredSeqs, docSeqs...)
			}
		}

		if len(expiredDocIDs) == 0 {
			continue
		}
		// 分块批量删除：一次 DeleteMany 干掉一整块过期文档，远优于旧实现的逐文档 DeleteOne。
		// 注：这里是"先扫描判定、后删除"，扫描与删除之间若有新消息写入，新消息可能随整文档一起被删除；
		// 该接口是管理员触发的低频历史清理，且以 timestamp 为界本就允许清理，残余竞态可接受。
		for start := 0; start < len(expiredDocIDs); start += deleteDocChunkSize {
			end := min(start+deleteDocChunkSize, len(expiredDocIDs))
			if _, err := m.mod.DeleteMany(ctx, bson.M{"doc_id": bson.M{"$in": expiredDocIDs[start:end]}}); err != nil {
				return nil, err
			}
		}
		deleted[conversationID] = expiredSeqs
	}

	return deleted, nil
}

// MarkDeleteBySeqs 标记删除（将 userID 加入目标槽位的 del_list）
// 每个 doc 执行两次 Update：先把历史 null 的 del_list 规范化为空数组，再按 msg.seq 定位槽位追加 userID。
// 槽位不存在（消息尚未持久化 / 已被物理删除 / 稀疏占位）时自动跳过，重复标记因 $addToSet 幂等也不会报错。
func (m *defaultMsgModel) MarkDeleteBySeqs(ctx context.Context, userID string, conversationID string, seqs []int64) error {
	if len(seqs) == 0 {
		return nil
	}

	// key: docID  value: 该 doc 下需要标记删除的 seq 集合（去重，同一消息多次标记无意义）
	doc := &model.MsgDocModel{}
	docSeqMap := make(map[string]map[int64]struct{})
	for _, seq := range seqs {
		docID := doc.GetDocID(conversationID, seq)
		if _, ok := docSeqMap[docID]; !ok {
			docSeqMap[docID] = make(map[int64]struct{})
		}
		docSeqMap[docID][seq] = struct{}{}
	}

	// 按 docID 循环：每个 doc 先规范化历史数据，再按 msg.seq 定位目标槽位追加 userID
	for docID, seqSet := range docSeqMap {
		// 1) 兼容历史数据：早期槽位的 del_list 未加 omitempty，nil 会落库成 del_list: null，
		//    此时 $addToSet 报 "Cannot apply $addToSet to non-array field"，先规范化为空数组。
		//    e.msg 存在 => 只处理真实消息槽位（稀疏占位与已被物理删除的槽位是 null 元素，跳过它们，
		//    否则 $set 会报 "Cannot create field in element null"）；
		//    $exists + $eq null => 只命中"字段存在且为 null"，新数据该字段缺失，不会被无谓改写。
		normalizeOpts := options.UpdateMany().SetArrayFilters([]any{
			bson.M{"e.msg": bson.M{"$exists": true}, "e.del_list": bson.M{"$exists": true, "$eq": nil}},
		})
		if _, err := m.mod.UpdateMany(ctx, bson.M{"doc_id": docID},
			bson.M{"$set": bson.M{"msgs.$[e].del_list": bson.A{}}}, normalizeOpts); err != nil {
			return err
		}

		// 2) 只对"目标 seq 且真实存在"的槽位追加 userID：槽位不存在（消息尚未持久化 / 已被物理删除 /
		//    稀疏占位）时不会匹配，因此不会报错；$addToSet 幂等 => 已删除的消息重复标记同样不报错，
		//    无需额外查询"消息是否已删除"。
		seqList := make([]int64, 0, len(seqSet))
		for seq := range seqSet {
			seqList = append(seqList, seq)
		}
		addOpts := options.UpdateMany().SetArrayFilters([]any{
			bson.M{"e.msg": bson.M{"$exists": true}, "e.msg.seq": bson.M{"$in": seqList}},
		})
		if _, err := m.mod.UpdateMany(ctx, bson.M{"doc_id": docID},
			bson.M{"$addToSet": bson.M{"msgs.$[e].del_list": userID}}, addOpts); err != nil {
			return err
		}
	}
	return nil
}

func (m *defaultMsgModel) MarkReadBySeqRange(ctx context.Context, userID string, conversationID string, seqStart, seqEnd int64) error {
	filter := docIDPrefix(conversationID)
	update := bson.M{
		"$set": bson.M{
			"msgs.$[e].is_read": true,
		},
	}
	opts := options.UpdateMany().SetArrayFilters([]any{
		bson.M{
			"e.msg.seq":     bson.M{"$gte": seqStart, "$lte": seqEnd},
			"e.msg.recv_id": userID,
		},
	})
	_, err := m.mod.UpdateMany(ctx, filter, update, opts)
	return err
}

func (m *defaultMsgModel) MarkReadBySeqs(ctx context.Context, userID string, conversationID string, seqs []int64) error {
	filter := docIDPrefix(conversationID)
	update := bson.M{
		"$set": bson.M{
			"msgs.$[e].is_read": true,
		},
	}
	opts := options.UpdateMany().SetArrayFilters([]any{
		bson.M{
			"e.msg.seq":     bson.M{"$in": seqs},
			"e.msg.recv_id": userID,
		},
	})
	_, err := m.mod.UpdateMany(ctx, filter, update, opts)
	return err
}

func (m *defaultMsgModel) GetLastMessageSeqByTimestamp(ctx context.Context, conversationID string, timestamp int64) (int64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"doc_id": bson.M{
					"$regex": fmt.Sprintf("^%s:", conversationID),
				},
			},
		},
		{
			"$match": bson.M{
				"msgs.msg.send_time": bson.M{
					"$lte": timestamp,
				},
			},
		},
		{
			"$sort": bson.M{
				"_id": -1,
			},
		},
		{
			"$limit": 1,
		},
		{
			"$project": bson.M{
				"_id":                0,
				"doc_id":             1,
				"msgs.msg.send_time": 1,
				"msgs.msg.seq":       1,
			},
		},
	}
	var doc model.MsgDocModel
	if err := m.mod.Aggregate(ctx, &doc, pipeline); err != nil {
		return 0, err
	}
	if len(doc.Msgs) > 0 {
		return doc.Msgs[0].Msg.Seq, nil
	}
	return 0, nil
}

func (m *defaultMsgModel) SearchMessage(ctx context.Context, req *model.SearchMessageReq) ([]*model.MsgInfoModel, error) {
	if req == nil {
		return nil, nil
	}

	// ===== 第一步：构造预过滤条件，减少 $unwind 需要处理的文档数 =====
	preMatch := bson.M{}
	// send_id 与 recv_id 直接在 msgs[N].msg 中匹配（会用数组索引，预过滤可无），
	// 这里先对会话类型做文档级粗筛：若指定了 sessionType，
	// 只需匹配该文档数组中至少一条消息为该类型（MongoDB 数组点号可命中）
	if req.SessionType > 0 {
		preMatch["msgs.msg.session_type"] = req.SessionType
	}
	if req.ContentType > 0 {
		preMatch["msgs.msg.content_type"] = req.ContentType
	}
	if req.SendID != "" {
		preMatch["msgs.msg.send_id"] = req.SendID
	}
	if req.RecvID != "" {
		preMatch["msgs.msg.recv_id"] = req.RecvID
	}
	if req.GroupID != "" {
		preMatch["msgs.msg.group_id"] = req.GroupID
	}
	if req.SendTime > 0 {
		preMatch["msgs.msg.send_time"] = bson.M{"$gte": req.SendTime}
	}

	// ===== 第二步：$unwind + 细粒度匹配 =====
	pipeline := []bson.M{}
	if len(preMatch) > 0 {
		pipeline = append(pipeline, bson.M{"$match": preMatch})
	}
	pipeline = append(pipeline, bson.M{"$unwind": "$msgs"})

	msgMatch := bson.M{"msgs.msg": bson.M{"$ne": nil}}
	if req.SessionType > 0 {
		msgMatch["msgs.msg.session_type"] = req.SessionType
	}
	if req.ContentType > 0 {
		msgMatch["msgs.msg.content_type"] = req.ContentType
	}
	if req.SendID != "" {
		msgMatch["msgs.msg.send_id"] = req.SendID
	}
	if req.RecvID != "" {
		msgMatch["msgs.msg.recv_id"] = req.RecvID
	}
	if req.GroupID != "" {
		msgMatch["msgs.msg.group_id"] = req.GroupID
	}
	if req.SendTime > 0 {
		msgMatch["msgs.msg.send_time"] = bson.M{"$gte": req.SendTime}
	}
	pipeline = append(pipeline, bson.M{"$match": msgMatch})

	// ===== 第三步：排序 + 分页 =====
	pipeline = append(pipeline, bson.M{"$sort": bson.M{"msgs.msg.send_time": -1}})

	page := req.Pagination.Page
	pageSize := req.Pagination.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	skip := int64(page-1) * int64(pageSize)
	pipeline = append(pipeline, bson.M{"$skip": skip})
	pipeline = append(pipeline, bson.M{"$limit": int64(pageSize)})

	// ===== 第四步：投影（把 unwind 出来的 msgs 字段名保留，方便直接反序列化） =====
	pipeline = append(pipeline, bson.M{
		"$project": bson.M{
			"_id":  0,
			"msgs": 1,
		},
	})

	// ===== 执行聚合 =====
	type searchDoc struct {
		Msgs *model.MsgInfoModel `bson:"msgs"`
	}
	var docs []*searchDoc
	if err := m.mod.Aggregate(ctx, &docs, pipeline); err != nil {
		return nil, err
	}

	result := make([]*model.MsgInfoModel, 0, len(docs))
	for _, d := range docs {
		if d != nil && d.Msgs != nil && d.Msgs.Msg != nil {
			result = append(result, d.Msgs)
		}
	}
	return result, nil
}
