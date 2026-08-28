package msg

import (
	"context"
	"errors"
	"fmt"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrMsgNotFound = errors.New("message not found")
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
	// DeleteByTimestamp 按时间戳物理删除过期文档
	DeleteByTimestamp(ctx context.Context, conversationIDs []string, timestamp int64) error
	// 标记删除（将 userID 加入 msgs.N.del_list）
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

func NewMsgModel(mod *mon.Model) MsgModel {
	return &defaultMsgModel{
		mod: mod,
	}
}

// docIDPrefix 返回 conversationID 的 DocID 前缀正则匹配
func docIDPrefix(conversationID string) bson.M {
	return bson.M{"doc_id": bson.M{"$regex": "^" + conversationID + ":"}}
}

// Insert 向分桶文档中插入一条消息。
// 通过 Upsert 创建文档（若不存在），$set 指定数组索引位置的消息槽位。
func (m *defaultMsgModel) Insert(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
	doc := &model.MsgDocModel{}
	docID := doc.GetDocID(conversationID, msg.Seq)
	msgIndex := doc.GetMsgIndex(msg.Seq)

	msgInfo := &model.MsgInfoModel{
		Msg: msg,
	}

	filter := bson.M{"doc_id": docID}
	update := bson.M{
		"$set": bson.M{
			"msgs." + convert.ToString(msgIndex): msgInfo,
		},
		"$setOnInsert": bson.M{
			"doc_id": docID,
		},
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := m.mod.UpdateOne(ctx, filter, update, opts)
	return err
}

// BatchInsert 批量插入消息。按 seq 分组到不同文档，逐文档 Upsert。
func (m *defaultMsgModel) BatchInsert(ctx context.Context, conversationID string, msgs []*model.MsgDataModel) error {
	doc := &model.MsgDocModel{}
	for _, msg := range msgs {
		docID := doc.GetDocID(conversationID, msg.Seq)
		msgIndex := doc.GetMsgIndex(msg.Seq)

		msgInfo := &model.MsgInfoModel{Msg: msg}
		filter := bson.M{"doc_id": docID}
		update := bson.M{
			"$set": bson.M{
				"msgs." + convert.ToString(msgIndex): msgInfo,
			},
			"$setOnInsert": bson.M{
				"doc_id": docID,
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

// DeleteBySeq 物理删除消息：$unset 清空指定槽位
func (m *defaultMsgModel) DeleteBySeq(ctx context.Context, conversationID string, seqs []int64) error {
	doc := &model.MsgDocModel{}
	for _, seq := range seqs {
		docID := doc.GetDocID(conversationID, seq)
		msgIndex := doc.GetMsgIndex(seq)

		filter := bson.M{"doc_id": docID}
		update := bson.M{
			"$unset": bson.M{
				"msgs." + convert.ToString(msgIndex): "",
			},
		}
		if _, err := m.mod.UpdateOne(ctx, filter, update); err != nil {
			return err
		}
	}
	return nil
}

// DeleteByTimestamp 按时间戳物理删除过期文档：遍历文档，若所有消息都过期则整文档删除
func (m *defaultMsgModel) DeleteByTimestamp(ctx context.Context, conversationIDs []string, timestamp int64) error {
	for _, conversationID := range conversationIDs {
		var docs []*model.MsgDocModel
		if err := m.mod.Find(ctx, &docs, docIDPrefix(conversationID)); err != nil {
			return err
		}

		for _, doc := range docs {
			allExpired := false
			hasAnyMsg := false
			for _, info := range doc.Msgs {
				if info != nil && info.Msg != nil {
					hasAnyMsg = true
					if info.Msg.SendTime > timestamp {
						allExpired = false
						break
					}
					allExpired = true
				}
			}
			if hasAnyMsg && allExpired {
				if _, err := m.mod.DeleteOne(ctx, bson.M{"doc_id": doc.DocID}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// MarkDeleteBySeqs 标记删除（将 userID 加入 msgs.N.del_list）
// 每 doc 仅执行一次 UpdateOne：按 seq 数字索引直接定位槽位路径，批量 $addToSet 追加 userID。
func (m *defaultMsgModel) MarkDeleteBySeqs(ctx context.Context, userID string, conversationID string, seqs []int64) error {
	// doc := &model.MsgDocModel{}
	// for _, seq := range seqs {
	// 	docID := doc.GetDocID(conversationID, seq)
	// 	msgIndex := doc.GetMsgIndex(seq)

	// 	filter := bson.M{"doc_id": docID}
	// 	update := bson.M{
	// 		"$addToSet": bson.M{
	// 			"msgs." + convert.ToString(msgIndex) + ".del_list": userID,
	// 		},
	// 	}
	// 	if _, err := m.mod.UpdateOne(ctx, filter, update); err != nil {
	// 		return err
	// 	}
	// }
	// return nil
	if len(seqs) == 0 {
		return nil
	}

	// key: docID  value: 该 doc 下需要标记删除的 msgIndex 列表（去重，同槽位多次写入无意义）
	doc := &model.MsgDocModel{}
	docIndexMap := make(map[string]map[int64]struct{})
	for _, seq := range seqs {
		docID := doc.GetDocID(conversationID, seq)
		idx := doc.GetMsgIndex(seq)
		if _, ok := docIndexMap[docID]; !ok {
			docIndexMap[docID] = make(map[int64]struct{})
		}
		docIndexMap[docID][idx] = struct{}{}
	}

	// 按 docID 循环，一个 doc 只执行一次 UpdateOne
	for docID, indexSet := range docIndexMap {
		addToSet := bson.M{}
		for idx := range indexSet {
			addToSet["msgs."+convert.ToString(idx)+".del_list"] = userID
		}

		update := bson.M{"$addToSet": addToSet}

		res, err := m.mod.UpdateMany(ctx, bson.M{"doc_id": docID}, update)
		if err != nil {
			return err
		}

		// 文档不存在，消息块还未创建，标记删除无法落地，打 warn 日志
		if res.MatchedCount == 0 {
			// log.Warnf("MarkDeleteBySeqs doc not exist, docID=%s indexes=%v", docID, indexSet)
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
