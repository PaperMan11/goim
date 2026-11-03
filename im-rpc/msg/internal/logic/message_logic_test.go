package logic

import (
	"context"
	"errors"
	"testing"

	"github.com/PaperMan11/goim/im-rpc/msg/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/authverify"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/queue"
	convservice "github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	userservice "github.com/PaperMan11/goim/pkg/rpcclient/userservice"
	"github.com/PaperMan11/goim/pkg/storage/model"
	msgModel "github.com/PaperMan11/goim/pkg/storage/mongo/msg"
	seqConversationModel "github.com/PaperMan11/goim/pkg/storage/mongo/seqconversation"
	seqUserModel "github.com/PaperMan11/goim/pkg/storage/mongo/sequser"
	allocator "github.com/PaperMan11/goim/pkg/storage/redis/allocator"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// ---- 测试桩：内嵌接口，仅覆盖同步提交路径实际用到的方法 ----

type stubAllocator struct {
	allocator.SeqAllocator
	allocate func(ctx context.Context, conversationID string) (int64, error)
}

func (s *stubAllocator) Allocate(ctx context.Context, conversationID string) (int64, error) {
	return s.allocate(ctx, conversationID)
}

type stubMsgModel struct {
	msgModel.MsgModel
	insertFn            func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error
	findByClientMsgIDFn func(ctx context.Context, conversationID string, clientMsgID string) (*model.MsgDataModel, error)
}

func (s *stubMsgModel) Insert(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
	return s.insertFn(ctx, conversationID, msg)
}

func (s *stubMsgModel) FindByClientMsgID(ctx context.Context, conversationID string, clientMsgID string) (*model.MsgDataModel, error) {
	return s.findByClientMsgIDFn(ctx, conversationID, clientMsgID)
}

type stubSeqConversationModel struct {
	seqConversationModel.SeqConversationModel
	setMaxSeq func(ctx context.Context, conversationID string, maxSeq int64) error
}

func (s *stubSeqConversationModel) SetConversationMaxSeq(ctx context.Context, conversationID string, maxSeq int64) error {
	return s.setMaxSeq(ctx, conversationID, maxSeq)
}

type stubSeqUserModel struct {
	seqUserModel.SeqUserModel
	setReadSeq func(ctx context.Context, userID string, conversationID string, readSeq int64) error
}

func (s *stubSeqUserModel) SetUserReadSeq(ctx context.Context, userID string, conversationID string, readSeq int64) error {
	return s.setReadSeq(ctx, userID, conversationID, readSeq)
}

type stubProducer struct {
	queue.Producer
	pushWithKey func(ctx context.Context, key, value string) error
}

func (s *stubProducer) PushWithKey(ctx context.Context, key, value string) error {
	return s.pushWithKey(ctx, key, value)
}

// ---- 构造工具 ----

func newTestLogic(t *testing.T, mutate func(*svc.ServiceContext)) *Logic {
	t.Helper()
	svcCtx := &svc.ServiceContext{
		// SendMsg 前置校验需要鉴权服务，默认放行
		AuthVerifier: &stubAuthVerifier{},
		// 免打扰校验：全局默认接收，会话不存在视为放行
		UserService: &stubUserService{},
		ConvService: &stubConvService{},
	}
	if mutate != nil {
		mutate(svcCtx)
	}
	return NewLogic(context.Background(), svcCtx)
}

type stubAuthVerifier struct {
	authverify.AuthVerifyService
}

func (s *stubAuthVerifier) CheckAccess(ctx context.Context, userID, opUserID string) (bool, error) {
	return true, nil
}

type stubUserService struct {
	userservice.UserService
}

func (s *stubUserService) GetGlobalRecvMessageOpt(ctx context.Context, req *pbuser.GetGlobalRecvMessageOptReq, _ ...grpc.CallOption) (*pbuser.GetGlobalRecvMessageOptResp, error) {
	return &pbuser.GetGlobalRecvMessageOptResp{GlobalRecvMsgOpt: constant.ReceiveMessage}, nil
}

type stubConvService struct {
	convservice.ConversationService
}

func (s *stubConvService) GetConversation(ctx context.Context, req *pbconv.GetConversationReq, _ ...grpc.CallOption) (*pbconv.GetConversationResp, error) {
	// 会话不存在 → 没有会话级免打扰覆盖，放行
	return nil, status.Error(codes.NotFound, "conversation not found")
}

func newSendMsg(clientMsgID string) *sdkws.MsgData {
	// 单聊会话 ID 按双方 ID 排序拼接：si_a_b
	return &sdkws.MsgData{
		ClientMsgID: clientMsgID,
		SendID:      "a",
		RecvID:      "b",
		SessionType: constant.SingleChatType,
		MsgFrom:     constant.UserMsgType,
		ContentType: constant.Text,
		Options:     map[string]bool{},
	}
}

// ---- 用例 ----

// 正常发送：分配 seq → 落库 → 游标推进 → sender ReadSeq → push 携带 seq（G3/G4）
func TestCommitUserMsgSync_Success(t *testing.T) {
	const allocatedSeq = 101
	var pushSeq int64
	var readSeqSet int64
	var cursorSet int64

	l := newTestLogic(t, func(sc *svc.ServiceContext) {
		sc.SeqAllocator = &stubAllocator{allocate: func(ctx context.Context, conversationID string) (int64, error) {
			return allocatedSeq, nil
		}}
		sc.MsgModel = &stubMsgModel{
			insertFn: func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
				return nil
			},
		}
		sc.SeqConversationModel = &stubSeqConversationModel{setMaxSeq: func(ctx context.Context, conversationID string, maxSeq int64) error {
			cursorSet = maxSeq
			return nil
		}}
		sc.SeqUserModel = &stubSeqUserModel{setReadSeq: func(ctx context.Context, userID string, conversationID string, readSeq int64) error {
			readSeqSet = readSeq
			return nil
		}}
		sc.MsgPushProducer = &stubProducer{pushWithKey: func(ctx context.Context, key, value string) error {
			resp := &pbmsg.PushMsgDataToMQ{}
			if err := proto.Unmarshal([]byte(value), resp); err != nil {
				t.Fatalf("unmarshal push payload: %v", err)
			}
			pushSeq = resp.GetMsgData().GetSeq()
			return nil
		}}
	})

	msgData := newSendMsg("client-1")
	if err := l.commitUserMsgSync(context.Background(), "single_sender_receiver", msgData); err != nil {
		t.Fatalf("commit user msg sync: %v", err)
	}
	if msgData.Seq != allocatedSeq {
		t.Fatalf("msgData.Seq = %d, want %d", msgData.Seq, allocatedSeq)
	}
	if cursorSet != allocatedSeq {
		t.Fatalf("cursor maxSeq = %d, want %d", cursorSet, allocatedSeq)
	}
	if readSeqSet != allocatedSeq {
		t.Fatalf("sender readSeq = %d, want %d", readSeqSet, allocatedSeq)
	}
	if pushSeq != allocatedSeq {
		t.Fatalf("push payload seq = %d, want %d", pushSeq, allocatedSeq)
	}
}

// 重复 clientMsgID：插入命中唯一索引冲突（E11000）后按幂等键读回，返回已有 seq（V4）
func TestCommitUserMsgSync_DuplicateClientMsgID(t *testing.T) {
	const (
		allocatedSeq = 202
		existingSeq  = 101
	)
	insertCalls := 0

	l := newTestLogic(t, func(sc *svc.ServiceContext) {
		sc.SeqAllocator = &stubAllocator{allocate: func(ctx context.Context, conversationID string) (int64, error) {
			return allocatedSeq, nil
		}}
		sc.MsgModel = &stubMsgModel{
			insertFn: func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
				insertCalls++
				// 模拟 uniq_client_msg_id 唯一索引冲突
				return mongo.CommandError{Code: 11000, Message: "E11000 duplicate key"}
			},
			findByClientMsgIDFn: func(ctx context.Context, conversationID string, clientMsgID string) (*model.MsgDataModel, error) {
				return &model.MsgDataModel{ClientMsgID: clientMsgID, Seq: existingSeq, ServerMsgID: "server-1"}, nil
			},
		}
		sc.SeqConversationModel = &stubSeqConversationModel{setMaxSeq: func(ctx context.Context, conversationID string, maxSeq int64) error {
			return nil
		}}
		sc.SeqUserModel = &stubSeqUserModel{setReadSeq: func(ctx context.Context, userID string, conversationID string, readSeq int64) error {
			return nil
		}}
		sc.MsgPushProducer = &stubProducer{pushWithKey: func(ctx context.Context, key, value string) error {
			return nil
		}}
	})

	msgData := newSendMsg("client-dup")
	if err := l.commitUserMsgSync(context.Background(), "single_sender_receiver", msgData); err != nil {
		t.Fatalf("duplicate send should succeed with existing seq: %v", err)
	}
	if msgData.Seq != existingSeq {
		t.Fatalf("msgData.Seq = %d, want existing %d", msgData.Seq, existingSeq)
	}
	if msgData.ServerMsgID != "server-1" {
		t.Fatalf("msgData.ServerMsgID = %q, want server-1", msgData.ServerMsgID)
	}
	if insertCalls != 1 {
		t.Fatalf("insert called %d times, want 1", insertCalls)
	}
}

// Mongo 停写：落库失败返回 90002 DatabaseError，发送方显式可见可重发（V2）
func TestCommitUserMsgSync_MongoDown(t *testing.T) {
	l := newTestLogic(t, func(sc *svc.ServiceContext) {
		sc.SeqAllocator = &stubAllocator{allocate: func(ctx context.Context, conversationID string) (int64, error) {
			return 303, nil
		}}
		sc.MsgModel = &stubMsgModel{
			insertFn: func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
				return errors.New("connection refused")
			},
		}
		sc.MsgPushProducer = &stubProducer{pushWithKey: func(ctx context.Context, key, value string) error {
			t.Fatal("push must not be called when persist fails")
			return nil
		}}
	})

	msgData := newSendMsg("client-err")
	err := l.commitUserMsgSync(context.Background(), "single_sender_receiver", msgData)
	if err == nil {
		t.Fatal("want error when insert fails")
	}
	if got := errx.ParseError(err).Code; got != errx.ErrCodeDatabaseError {
		t.Fatalf("errCode = %d, want %d (DatabaseError)", got, errx.ErrCodeDatabaseError)
	}
}

// seq 分配失败：返回 90002，且不触发落库与推送
func TestCommitUserMsgSync_AllocateFailed(t *testing.T) {
	l := newTestLogic(t, func(sc *svc.ServiceContext) {
		sc.SeqAllocator = &stubAllocator{allocate: func(ctx context.Context, conversationID string) (int64, error) {
			return 0, errors.New("redis pool exhausted")
		}}
		sc.MsgModel = &stubMsgModel{
			insertFn: func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
				t.Fatal("insert must not be called when allocate fails")
				return nil
			},
		}
	})

	msgData := newSendMsg("client-alloc")
	err := l.commitUserMsgSync(context.Background(), "single_sender_receiver", msgData)
	if err == nil {
		t.Fatal("want error when allocate fails")
	}
	if got := errx.ParseError(err).Code; got != errx.ErrCodeDatabaseError {
		t.Fatalf("errCode = %d, want %d (DatabaseError)", got, errx.ErrCodeDatabaseError)
	}
}

// 游标推进失败：返回 90002，保证 ack ⇒ 游标已推进（G1）
func TestCommitUserMsgSync_CursorAdvanceFailed(t *testing.T) {
	l := newTestLogic(t, func(sc *svc.ServiceContext) {
		sc.SeqAllocator = &stubAllocator{allocate: func(ctx context.Context, conversationID string) (int64, error) {
			return 404, nil
		}}
		sc.MsgModel = &stubMsgModel{
			insertFn: func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
				return nil
			},
		}
		sc.SeqConversationModel = &stubSeqConversationModel{setMaxSeq: func(ctx context.Context, conversationID string, maxSeq int64) error {
			return errors.New("mongo timeout")
		}}
		sc.MsgPushProducer = &stubProducer{pushWithKey: func(ctx context.Context, key, value string) error {
			t.Fatal("push must not be called when cursor advance fails")
			return nil
		}}
	})

	msgData := newSendMsg("client-cursor")
	err := l.commitUserMsgSync(context.Background(), "single_sender_receiver", msgData)
	if err == nil {
		t.Fatal("want error when cursor advance fails")
	}
	if got := errx.ParseError(err).Code; got != errx.ErrCodeDatabaseError {
		t.Fatalf("errCode = %d, want %d (DatabaseError)", got, errx.ErrCodeDatabaseError)
	}
}

// 冲突读回也失败（如读回超时）：必须显式报错，不能静默按新消息处理
func TestCommitUserMsgSync_ReadBackFailed(t *testing.T) {
	l := newTestLogic(t, func(sc *svc.ServiceContext) {
		sc.SeqAllocator = &stubAllocator{allocate: func(ctx context.Context, conversationID string) (int64, error) {
			return 505, nil
		}}
		sc.MsgModel = &stubMsgModel{
			insertFn: func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
				return mongo.CommandError{Code: 11000, Message: "E11000 duplicate key"}
			},
			findByClientMsgIDFn: func(ctx context.Context, conversationID string, clientMsgID string) (*model.MsgDataModel, error) {
				return nil, errors.New("read back timeout")
			},
		}
		sc.MsgPushProducer = &stubProducer{pushWithKey: func(ctx context.Context, key, value string) error {
			return nil
		}}
	})

	msgData := newSendMsg("client-readback")
	err := l.commitUserMsgSync(context.Background(), "single_sender_receiver", msgData)
	if err == nil {
		t.Fatal("want error when read back fails")
	}
	if got := errx.ParseError(err).Code; got != errx.ErrCodeDatabaseError {
		t.Fatalf("errCode = %d, want %d (DatabaseError)", got, errx.ErrCodeDatabaseError)
	}
}

// SendMsg resp 契约：同步提交后 resp 携带 seq/conversationID（G4）
func TestSendMsgRespCarriesSeqAndConversationID(t *testing.T) {
	const allocatedSeq = 606

	l := newTestLogic(t, func(sc *svc.ServiceContext) {
		sc.SeqAllocator = &stubAllocator{allocate: func(ctx context.Context, conversationID string) (int64, error) {
			return allocatedSeq, nil
		}}
		sc.MsgModel = &stubMsgModel{
			insertFn: func(ctx context.Context, conversationID string, msg *model.MsgDataModel) error {
				return nil
			},
		}
		sc.SeqConversationModel = &stubSeqConversationModel{setMaxSeq: func(ctx context.Context, conversationID string, maxSeq int64) error {
			return nil
		}}
		sc.SeqUserModel = &stubSeqUserModel{setReadSeq: func(ctx context.Context, userID string, conversationID string, readSeq int64) error {
			return nil
		}}
		sc.MsgPushProducer = &stubProducer{pushWithKey: func(ctx context.Context, key, value string) error {
			return nil
		}}
	})

	resp, err := l.SendMsg(context.Background(), &pbmsg.SendMsgReq{MsgData: newSendMsg("client-resp")})
	if err != nil {
		t.Fatalf("SendMsg: %v", err)
	}
	if resp.GetSeq() != allocatedSeq {
		t.Fatalf("resp.Seq = %d, want %d", resp.GetSeq(), allocatedSeq)
	}
	if want := "si_a_b"; resp.GetConversationID() != want {
		t.Fatalf("resp.ConversationID = %q, want %q", resp.GetConversationID(), want)
	}
}
