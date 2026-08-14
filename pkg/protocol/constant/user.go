// Package constant 定义了 OpenIM 系统中使用的各种常量
package constant

// 用户角色和权限常量
const (
	// 用户类型
	AppSuperAdmin        = 9999 // 应用超级管理员
	IMOrdinaryUser       = 0    // IM 普通用户
	AppOrdinaryUsers     = 1    // 应用普通用户
	AppAdmin             = 2    // 应用管理员
	AppNotificationAdmin = 3    // 应用通知管理员
	AppRobotAdmin        = 4    // 应用机器人管理员

	// 好友申请响应状态
	FriendResponseNotHandle = 0  // 未处理
	FriendResponseAgree     = 1  // 同意
	FriendResponseRefuse    = -1 // 拒绝

	// 性别
	Male   = 1 // 男性
	Female = 2 // 女性

	// 在线状态
	Online  = 1 // 在线
	Offline = 0 // 离线

	// 注册状态
	Registered   = 1 // 已注册
	UnRegistered = 0 // 未注册

	// 用户订阅相关常量
	SubscriberUser = 1 // 订阅用户
	Unsubscribe    = 2 // 取消订阅
)

// 好友接受提示语
const FriendAcceptTip = "You have successfully become friends, so start chatting"

// 好友添加方式
const (
	BecomeFriendByImport = 1 // 通过管理员导入成为好友
	BecomeFriendByApply  = 2 // 通过申请成为好友
)
