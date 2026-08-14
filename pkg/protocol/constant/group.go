// Package constant 定义了 OpenIM 系统中使用的各种常量
package constant

// 群组状态
const (
	GroupOk              = 0 // 正常
	GroupBanChat         = 1 // 禁止发言
	GroupStatusDismissed = 2 // 已解散
	GroupStatusMuted     = 3 // 已禁言
)

// 群组类型
const (
	NormalGroup  = 0 // 普通群
	SuperGroup   = 1 // 超级群
	WorkingGroup = 2 // 工作群
)

const (
	GroupBaned          = 3 // 已封禁
	GroupBanPrivateChat = 4 // 禁止私聊
)

// 群角色
const (
	GroupOwner         = 100 // 群主
	GroupAdmin         = 60  // 群管理员
	GroupOrdinaryUsers = 20  // 群普通成员
)

// 群组申请响应状态
const (
	GroupResponseAgree  = 1  // 同意
	GroupResponseRefuse = -1 // 拒绝
)

// 用户加入群组的来源
const (
	JoinByAdmin      = 1 // 管理员添加
	JoinByInvitation = 2 // 邀请加入
	JoinBySearch     = 3 // 搜索加入
	JoinByQRCode     = 4 // 扫码加入
)

// 群组加入验证模式
const (
	ApplyNeedVerificationInviteDirectly = 0 // 申请需要验证，邀请直接加入
	AllNeedVerification                 = 1 // 所有人都需要验证 (除非群主或管理员邀请)
	Directly                            = 2 // 直接加入，无需验证
)

// 群组 RPC 消息批量大小
const (
	GroupRPCRecvSize = 30 // 群组 RPC 接收批量大小
	GroupRPCSendSize = 30 // 群组 RPC 发送批量大小
)

// 群组搜索位置
const (
	GroupSearchPositionHead = 1 // 搜索位置在头部
	GroupSearchPositionAny  = 2 // 搜索位置在任意位置
)

// GroupIsBanChat 检查群组是否被禁言
// 参数 status: 群组状态
// 返回：如果群组被禁言返回 true，否则返回 false
func GroupIsBanChat(status int32) bool {
	if status != GroupStatusMuted {
		return false
	}
	return true
}

// GroupIsBanPrivateChat 检查是否禁止私聊
// 参数 status: 群组状态
// 返回：如果禁止私聊返回 true，否则返回 false
func GroupIsBanPrivateChat(status int32) bool {
	if status != GroupBanPrivateChat {
		return false
	}
	return true
}
