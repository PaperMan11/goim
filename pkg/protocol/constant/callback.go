// Package constant 定义了 OpenIM 系统中使用的各种常量
package constant

// 回调命令
const (
	CallbackBeforeSendSingleMsgCommand                   = "callbackBeforeSendSingleMsgCommand"                   // 发送单聊消息前回调
	CallbackAfterSendSingleMsgCommand                    = "callbackAfterSendSingleMsgCommand"                    // 发送单聊消息后回调
	CallbackBeforeSendGroupMsgCommand                    = "callbackBeforeSendGroupMsgCommand"                    // 发送群消息前回调
	CallbackAfterSendGroupMsgCommand                     = "callbackAfterSendGroupMsgCommand"                     // 发送群消息后回调
	CallbackMsgModifyCommand                             = "callbackMsgModifyCommand"                             // 消息修改回调
	CallbackUserOnlineCommand                            = "callbackUserOnlineCommand"                            // 用户上线回调
	CallbackUserOfflineCommand                           = "callbackUserOfflineCommand"                           // 用户下线回调
	CallbackUserKickOffCommand                           = "callbackUserKickOffCommand"                           // 用户被踢回调
	CallbackOfflinePushCommand                           = "callbackOfflinePushCommand"                           // 离线推送回调
	CallbackOnlinePushCommand                            = "callbackOnlinePushCommand"                            // 在线推送回调
	CallbackSuperGroupOnlinePushCommand                  = "callbackSuperGroupOnlinePushCommand"                  // 超级群在线推送回调
	CallbackBeforeAddFriendCommand                       = "callbackBeforeAddFriendCommand"                       // 添加好友前回调
	CallbackBeforeUpdateUserInfoCommand                  = "callbackBeforeUpdateUserInfoCommand"                  // 更新用户信息前回调
	CallbackBeforeCreateGroupCommand                     = "callbackBeforeCreateGroupCommand"                     // 创建群前回调
	CallbackBeforeMemberJoinGroupCommand                 = "callbackBeforeMemberJoinGroupCommand"                 // 成员加群前回调
	CallbackBeforeSetGroupMemberInfoCommand              = "CallbackBeforeSetGroupMemberInfoCommand"              // 设置群成员信息前回调
	CallbackBeforeSetMessageReactionExtensionCommand     = "callbackBeforeSetMessageReactionExtensionCommand"     // 设置消息表情回应前回调
	CallbackBeforeDeleteMessageReactionExtensionsCommand = "callbackBeforeDeleteMessageReactionExtensionsCommand" // 删除消息表情回应前回调
	CallbackGetMessageListReactionExtensionsCommand      = "callbackGetMessageListReactionExtensionsCommand"      // 获取消息列表表情回应回调
	CallbackAddMessageListReactionExtensionsCommand      = "callbackAddMessageListReactionExtensionsCommand"      // 添加消息列表表情回应回调
)

// 回调动作代码
const (
	ActionAllow     = 0 // 允许
	ActionForbidden = 1 // 禁止
)

// 回调处理代码
const (
	CallbackHandleSuccess = 0 // 处理成功
	CallbackHandleFailed  = 1 // 处理失败
)

// 回调命令键
const CallbackCommand = "command"
