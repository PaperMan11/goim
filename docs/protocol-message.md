# 消息协议文档（各场景消息格式）

本文档定义 GoIM 系统中消息的统一结构 `MsgData`、内容类型（contentType）枚举，以及**各种场景下 `content` 字段的具体 JSON 格式**。

协议定义代码位置：

- MsgData 与各通知结构：[pkg/protocol/sdkws/sdkws.proto](file:///c:/Users/Administrator/Desktop/goim/pkg/protocol/sdkws/sdkws.proto)
- 内容类型常量：[pkg/protocol/constant/content_type.go](file:///c:/Users/Administrator/Desktop/goim/pkg/protocol/constant/content_type.go)
- 会话类型常量：[pkg/protocol/constant/conversation.go](file:///c:/Users/Administrator/Desktop/goim/pkg/protocol/constant/conversation.go)
- 消息选项常量：[pkg/protocol/constant/msg_option.go](file:///c:/Users/Administrator/Desktop/goim/pkg/protocol/constant/msg_option.go)
- 会话 ID 生成规则：[pkg/msgprocessor/conversation.go](file:///c:/Users/Administrator/Desktop/goim/pkg/msgprocessor/conversation.go)
- 通知内容封装：[pkg/msgdispatcher/dispatcher.go](file:///c:/Users/Administrator/Desktop/goim/pkg/msgdispatcher/dispatcher.go)

---

## 1. 消息统一结构 MsgData

所有消息（普通消息、通知消息）在网络与存储层统一使用 `sdkws.MsgData`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `sendID` | string | 发送者 ID（系统通知为系统账号，如 `system`） |
| `recvID` | string | 接收者 ID（单聊/通知时使用） |
| `groupID` | string | 群组 ID（群消息时使用，此时 recvID 为空） |
| `clientMsgID` | string | 客户端消息 ID（客户端生成 UUID，用于去重/撤回定位） |
| `serverMsgID` | string | 服务端消息 ID（服务端生成） |
| `senderPlatformID` | int | 发送者平台 ID（见附录 A） |
| `senderNickname` | string | 发送者昵称 |
| `senderFaceURL` | string | 发送者头像 URL |
| `sessionType` | int | 会话类型：`1`单聊 / `2`写扩散群聊 / `3`读扩散群聊 / `4`通知会话 |
| `msgFrom` | int | 消息来源：`100`用户消息 / `200`系统消息 |
| `contentType` | int | 内容类型，见 §2 |
| `content` | string | 消息内容（JSON 字符串，格式随 contentType 而定，见 §3/§4） |
| `seq` | int64 | 消息序列号（服务端分配，会话内单调递增） |
| `sendTime` | int64 | 发送时间戳（毫秒，服务端填充） |
| `createTime` | int64 | 创建时间戳（毫秒，客户端填充） |
| `status` | int | 消息状态：`1`发送中 / `2`发送成功 / `3`发送失败 / `4`已删除 / `5`已过滤 |
| `isRead` | bool | 是否已读 |
| `options` | map<string,bool> | 消息选项，见 §5 |
| `offlinePushInfo` | OfflinePushInfo | 离线推送信息（title/desc/ex/iOSPushSound/iOSBadgeCount/signalInfo） |
| `atUserIDList` | []string | 被 @ 用户 ID 列表（@所有人时为 `["AtAllTag"]`） |
| `attachedInfo` | string | 附加信息 |
| `ex` | string | 扩展字段 |

> 说明：`content` 为 string 类型，承载 UTF-8 JSON 文本；HTTP API 直接传 JSON 字符串即可（无需 Base64），内部 gRPC/Kafka 二进制传输与原 bytes 类型 wire 兼容。

### 1.1 会话 ID 规则

会话 ID 由服务端按规则生成，客户端拉取/推送中可见：

| 前缀 | 场景 | 格式 |
|---|---|---|
| `si_` | 单聊 | `si_<较小userID>_<较大userID>`（排序拼接） |
| `g_` | 写扩散群聊 | `g_<groupID>` |
| `sg_` | 读扩散群聊（超级群） | `sg_<groupID>` |
| `n_` | 通知 | `n_<sendID>_<recvID>` 或 `n_<groupID>` |
| `sn_` | 通知会话 | `sn_<较小userID>_<较大userID>` |

---

## 2. 内容类型 contentType

### 2.1 基础消息（用户发送）

| contentType | 常量 | 说明 |
|---|---|---|
| 101 | Text | 文本消息 |
| 102 | Picture | 图片消息 |
| 103 | Voice | 语音消息 |
| 104 | Video | 视频消息 |
| 105 | File | 文件消息 |
| 106 | AtText | @消息 |
| 107 | Merger | 合并转发消息 |
| 108 | Card | 名片消息 |
| 109 | Location | 位置消息 |
| 110 | Custom | 自定义消息 |
| 111 | Revoke | 撤回消息 |
| 113 | Typing | 正在输入 |
| 114 | Quote | 引用消息 |
| 117 | AdvancedText | 高级文本 |
| 118 | MarkdownText | Markdown 文本 |
| 119 | CustomNotTriggerConversation | 不触发会话的自定义消息 |
| 120 | CustomOnlineOnly | 仅在线可见的自定义消息 |
| 121 | ReactionMessageModifier | 表情回应修改 |
| 122 | ReactionMessageDeleter | 表情回应删除 |

### 2.2 通用消息类型

| contentType | 常量 | 说明 |
|---|---|---|
| 200 | Common | 普通消息 |
| 201 | GroupMsg | 群消息 |
| 202 | SignalMsg | 信令消息（RTC 音视频通话信令） |
| 203 | CustomNotification | 自定义通知 |

### 2.3 通知消息（系统产生，≥1000）

| contentType | 常量 | detail 结构（proto） |
|---|---|---|
| 1201 | FriendApplicationApprovedNotification | FriendApplicationApprovedTips |
| 1202 | FriendApplicationRejectedNotification | FriendApplicationRejectedTips |
| 1203 | FriendApplicationNotification | FriendApplicationTips |
| 1204 | FriendAddedNotification | FriendAddedTips |
| 1205 | FriendDeletedNotification | FriendDeletedTips |
| 1206 | FriendRemarkSetNotification | FriendInfoChangedTips |
| 1207 | BlackAddedNotification | BlackAddedTips |
| 1208 | BlackDeletedNotification | BlackDeletedTips |
| 1209 | FriendInfoUpdatedNotification | FriendInfoChangedTips |
| 1210 | FriendsInfoUpdateNotification | FriendsInfoUpdateTips |
| 1300 | ConversationChangeNotification | ConversationUpdateTips |
| 1303 | UserInfoUpdatedNotification | UserInfoUpdatedTips |
| 1304 | UserStatusChangeNotification | UserStatusChangeTips |
| 1305 | UserCommandAddNotification | UserCommandAddTips |
| 1306 | UserCommandDeleteNotification | UserCommandDeleteTips |
| 1307 | UserCommandUpdateNotification | UserCommandUpdateTips |
| 1308 | UserSubscribeOnlineStatusNotification | SubUserOnlineStatusTips |
| 1400 | OANotification | 自定义 |
| 1501 | GroupCreatedNotification | GroupCreatedTips |
| 1502 | GroupInfoSetNotification | GroupInfoSetTips |
| 1503 | JoinGroupApplicationNotification | JoinGroupApplicationTips |
| 1504 | MemberQuitNotification | MemberQuitTips |
| 1505 | GroupApplicationAcceptedNotification | GroupApplicationAcceptedTips |
| 1506 | GroupApplicationRejectedNotification | GroupApplicationRejectedTips |
| 1507 | GroupOwnerTransferredNotification | GroupOwnerTransferredTips |
| 1508 | MemberKickedNotification | MemberKickedTips |
| 1509 | MemberInvitedNotification | MemberInvitedTips |
| 1510 | MemberEnterNotification | MemberEnterTips |
| 1511 | GroupDismissedNotification | GroupDismissedTips |
| 1512 | GroupMemberMutedNotification | GroupMemberMutedTips |
| 1513 | GroupMemberCancelMutedNotification | GroupMemberCancelMutedTips |
| 1514 | GroupMutedNotification | GroupMutedTips |
| 1515 | GroupCancelMutedNotification | GroupCancelMutedTips |
| 1516 | GroupMemberInfoSetNotification | GroupMemberInfoSetTips |
| 1517 | GroupMemberSetToAdminNotification | GroupMemberInfoSetTips |
| 1518 | GroupMemberSetToOrdinaryUserNotification | GroupMemberInfoSetTips |
| 1519 | GroupInfoSetAnnouncementNotification | GroupInfoSetAnnouncementTips |
| 1520 | GroupInfoSetNameNotification | GroupInfoSetNameTips |
| 1651 | SuperGroupUpdateNotification | 自定义 |
| 1652 | MsgDeleteNotification | DeleteMessageTips |
| 1701 | ConversationPrivateChatNotification | ConversationSetPrivateTips |
| 1702 | ConversationUnreadNotification | ConversationUpdateTips |
| 1703 | ClearConversationNotification | ClearConversationTips |
| 1704 | ConversationDeleteNotification | ConversationDeleteTips |
| 2001 | BusinessNotification | 自定义（业务通知 2000-2099） |
| 2101 | MsgRevokeNotification | RevokeMsgTips |
| 2102 | DeleteMsgsNotification | DeleteMsgsTips |
| 2200 | HasReadReceipt | MarkAsReadTips |

---

## 3. 普通消息的 content 格式

以下 `content` 均为 **JSON 字符串**（外层 MsgData 中需转义）。

### 3.1 文本消息（101 Text）

```json
{"text":"Hello, 世界","atUsers":[]}
```

### 3.2 图片消息（102 Picture）

```json
{
  "origPictureElem":  {"url":"https://cdn.example.com/img/p=1","width":1920,"height":1080,"fileSize":500000},
  "bigPictureElem":   {"url":"https://cdn.example.com/img/p=1-big","width":3840,"height":2160},
  "snapShotElem":     {"url":"https://cdn.example.com/img/p=1-thumb","width":300,"height":300}
}
```

### 3.3 语音消息（103 Voice）

```json
{
  "url":"https://cdn.example.com/voice/v=1",
  "duration":3840,
  "baseInfo":{"url":"https://cdn.example.com/voice/v=1-base","duration":3840,"size":60000}
}
```

`duration` 单位：毫秒。

### 3.4 视频消息（104 Video）

```json
{
  "videoPath":"https://cdn.example.com/video/v=1",
  "videoURL":"https://cdn.example.com/video/v=1-out",
  "snapshotURL":"https://cdn.example.com/video/v=1-snap",
  "duration":15000,
  "baseInfo":{"url":"https://cdn.example.com/video/v=1-base","duration":15000,"size":2000000}
}
```

### 3.5 文件消息（105 File）

```json
{
  "filePath":"https://cdn.example.com/file/f=1",
  "fileName":"design.pdf",
  "fileSize":500000,
  "baseInfo":{"url":"https://cdn.example.com/file/f=1-base","size":500000}
}
```

### 3.6 @消息（106 AtText）

@单人：

```json
{
  "text":"@user_002 加油吧",
  "atUsers":[{"atUserID":"user_002","subTop":"user_002"}]
}
```

@所有人（`atUserID` 固定为 `AtAllTag`，同时 `MsgData.atUserIDList=["AtAllTag"]`）：

```json
{
  "text":"@AtAllTag 本周技术分享改到周五下午",
  "atUsers":[{"atUserID":"AtAllTag","subTop":"AtAllTag"}]
}
```

### 3.7 合并转发消息（107 Merger）

```json
{
  "title":"历史聊天记录:2条",
  "isCollapseBirthdayTime":0,
  "messageEntityList":[
    {"senderNickname":"Bob","senderID":"user_002","text":"上季 go-conference","contentType":101,"time":1650000000000},
    {"senderNickname":"Alice","senderID":"user_001","text":"确认线下交付","contentType":101,"time":1650000000001}
  ],
  "nicknameList":["Bob","Alice"],
  "platformIDList":[1,2]
}
```

### 3.8 名片消息（108 Card）

```json
{"userID":"user_002","nickname":"Bob","faceURL":"https://cdn.example.com/u/002.png","data":""}
```

### 3.9 位置消息（109 Location）

```json
{
  "description":"西湖区学院路文三路口",
  "longitude":120.2036,
  "latitude":30.2594,
  "baseInfo":{"longitude":120.2036,"latitude":30.2594,"description":"西子湖畔杭州城"}
}
```

### 3.10 自定义消息（110 Custom）

```json
{"data":"{\"key\":\"value\"}","extension":"","description":""}
```

- `data`：业务自定义负载（通常为 JSON 字符串）；
- `extension`：扩展字段；`description`：描述文本。

### 3.11 撤回消息（111 Revoke）

`content` 为撤回描述信息：

```json
{
  "clientMsgID":"client-uuid-0001",
  "revokerID":"user_001",
  "revokerNickname":"Alice",
  "revokerRole":0,
  "revokeTime":0,
  "sourceMessageSendID":"user_001",
  "sourceMessageSenderNickname":"Alice",
  "sessionType":1,
  "seq":1
}
```

同时系统会向相关端发送 2101 撤回通知（见 §4.5）。

### 3.12 正在输入（113 Typing）

```json
{"msgContent":"1"}
```

约定 `options` 全部关闭（history/persistent/offlinePush/unreadCount/conversationUpdate 均为 false），不落库、不计未读。

### 3.13 引用消息（114 Quote）

```json
{
  "text":"已确认线下交付方案",
  "quoteMessage":{"text":"上季讨论的接入层方案？","elmType":101},
  "atUsers":[]
}
```

`quoteMessage.elmType` 为被引用消息的 contentType，`text` 为被引用消息的摘要文本。

### 3.14 高级文本（117 AdvancedText）

```json
{"text":"告，本周获最佳工程团队提案奖候选"}
```

### 3.15 Markdown 文本（118 MarkdownText）

```json
{"text":"## 周报\n- 在线人数峰值：100人"}
```

### 3.16 不触发会话的自定义消息（119 CustomNotTriggerConversation）

格式同 110。约定 `conversationUpdate=false`、`unreadCount=false`，仅透传不更新会话列表。

### 3.17 仅在线自定义消息（120 CustomOnlineOnly）

格式同 110。约定 `history=false`、`persistent=false`，接收方不在线则丢弃，常用于实时状态同步。

### 3.18 信令消息（202 SignalMsg）

`content` 为 RTC 信令 JSON（音视频通话邀请/应答/挂断等），具体结构由 RTC 业务定义，例如：

```json
{"signalType":"invite","roomID":"r-1001","mediaType":"video","ex":""}
```

### 3.19 自定义通知（203 CustomNotification）

```json
{"serviceName":"platform_login","data":"{\"service\":\"platform_login\",\"time\":1650000000000}"}
```

---

## 4. 通知消息的 content 格式

### 4.1 统一封装

通知消息（contentType ≥ 1000）的 `content` 统一为 `NotificationElem` 的 JSON：

```json
{"detail":"<Tips 结构的 JSON 字符串>"}
```

注意 `detail` 是**字符串内嵌 JSON**（二次编码），解析时需先解析外层取 `detail`，再对 `detail` 做一次 JSON 解析得到 Tips 对象。

通知消息公共特征：`msgFrom=200`（系统消息）、`sendID` 为系统/操作账号；可靠性级别（UnreliableNotification=1 / ReliableNotificationNoMsg=2 / ReliableNotificationMsg=3）决定是否落库，可靠级别通知客户端可通过 1002 拉取补齐。

### 4.2 好友通知（1201-1210）

**1203 收到好友申请** — detail 为 FriendApplicationTips：

```json
{
  "fromToUserID":{"fromUserID":"user_001","toUserID":"user_002"},
  "request":{
    "fromUserID":"user_001","fromNickname":"Alice","fromFaceURL":"https://...",
    "toUserID":"user_002","toNickname":"Bob","toFaceURL":"https://...",
    "handleResult":0,"reqMsg":"交个朋友","createTime":1650000000000,
    "handlerUserID":"","handleMsg":"","handleTime":0,"ex":""
  }
}
```

**1201 好友申请已同意** — detail 为 FriendApplicationApprovedTips：

```json
{
  "fromToUserID":{"fromUserID":"user_002","toUserID":"user_001"},
  "handleMsg":"同意",
  "friendVersion":12,"friendVersionID":"65f1...",
  "request":{"...":"同 FriendRequest 结构"}
}
```

**1202 好友申请已拒绝** — detail 为 FriendApplicationRejectedTips（同上，无 version 字段）。

**1204 已添加好友** — detail 为 FriendAddedTips：

```json
{
  "friend":{"ownerUserID":"user_001","remark":"","createTime":1650000000000,
    "friendUser":{"userID":"user_002","nickname":"Bob","faceURL":"https://...","ex":"","createTime":0,"appMangerLevel":0,"globalRecvMsgOpt":0},
    "addSource":2,"operatorUserID":"user_001","ex":"","isPinned":false},
  "operationTime":1650000000000,
  "opUser":{"userID":"user_001","nickname":"Alice","faceURL":"https://...","ex":""},
  "friendVersion":13,"friendVersionID":"65f2..."
}
```

**1205 已删除好友 / 1206 好友备注已设置 / 1209 好友信息已更新** — detail 为 FriendDeletedTips / FriendInfoChangedTips：

```json
{"fromToUserID":{"fromUserID":"user_001","toUserID":"user_002"},"friendVersion":14,"friendVersionID":"65f3..."}
```

**1207 已加入黑名单 / 1208 已移出黑名单** — detail 为 BlackAddedTips / BlackDeletedTips：

```json
{"fromToUserID":{"fromUserID":"user_001","toUserID":"user_002"}}
```

**1210 好友信息批量更新** — detail 为 FriendsInfoUpdateTips：

```json
{"fromToUserID":{"fromUserID":"user_001","toUserID":"user_002"},"friendIDs":["user_003"],"friendVersion":15,"friendVersionID":"65f4..."}
```

### 4.3 用户通知（1303-1308）

**1303 用户信息已更新**：

```json
{"userID":"user_001"}
```

**1304 用户状态变更**（订阅了在线状态的用户收到）：

```json
{"fromUserID":"user_001","toUserID":"user_002","status":1,"platformID":5}
```

`status`：`1`在线 / `0`离线。

**1305/1306/1307 用户命令添加/删除/更新**：

```json
{"fromUserID":"user_001","toUserID":"user_002"}
```

**1308 订阅用户在线状态结果** — detail 为 SubUserOnlineStatusTips：

```json
{"subscribers":[{"userID":"user_003","onlinePlatformIDs":[5,3]}]}
```

### 4.4 群组通知（1501-1520）

以下 Tips 中公共子结构：

- `GroupInfo`：`{groupID,groupName,notification,introduction,faceURL,ownerUserID,createTime,memberCount,ex,status,creatorUserID,groupType,needVerification,lookMemberInfo,applyMemberFriend,notificationUpdateTime,notificationUserID}`
- `GroupMemberFullInfo`：`{groupID,userID,roleLevel,joinTime,nickname,faceURL,appMangerLevel,joinSource,operatorUserID,ex,muteEndTime,inviterUserID}`（`roleLevel`：100群主/60管理员/20普通成员；`joinSource`：1管理员添加/2邀请/3搜索/4扫码）

**1501 群组已创建** — GroupCreatedTips：

```json
{
  "group":{"...":"GroupInfo"},
  "opUser":{"...":"GroupMemberFullInfo"},
  "memberList":[{"...":"GroupMemberFullInfo"}],
  "operationTime":1650000000000,
  "groupOwnerUser":{"...":"GroupMemberFullInfo"},
  "groupMemberVersion":1,"groupMemberVersionID":"65f5..."
}
```

**1502 群组信息已设置** — GroupInfoSetTips：`{opUser,muteTime,group,groupMemberVersion,groupMemberVersionID}`

**1503 加群申请** — JoinGroupApplicationTips：

```json
{
  "group":{"...":"GroupInfo"},
  "applicant":{"userID":"user_009","nickname":"Carol","faceURL":"https://...","ex":""},
  "reqMsg":"申请加入",
  "uuid":"req-uuid-0001",
  "request":{"...":"GroupRequest"}
}
```

**1504 成员退出** — MemberQuitTips：`{group,quitUser,operationTime,groupMemberVersion,groupMemberVersionID}`

**1505 群申请已接受 / 1506 群申请已拒绝** — GroupApplicationAcceptedTips / GroupApplicationRejectedTips：

```json
{"group":{"...":"GroupInfo"},"opUser":{"...":"GroupMemberFullInfo"},"handleMsg":"欢迎","receiverAs":0,"uuid":"req-uuid-0001","request":{"...":"GroupRequest"}}
```

`receiverAs`：`1`发给管理员 / `0`发给申请人。

**1507 群主已转让** — GroupOwnerTransferredTips：

```json
{"group":{"...":"GroupInfo"},"opUser":{"...":"..."},"newGroupOwner":{"...":"..."},"oldGroupOwner":"user_001","operationTime":1650000000000,"oldGroupOwnerInfo":{"...":"..."},"groupMemberVersion":8,"groupMemberVersionID":"..."}
```

**1508 成员被踢** — MemberKickedTips：`{group,opUser,kickedUserList[],operationTime,groupMemberVersion,groupMemberVersionID}`

**1509 成员被邀请入群** — MemberInvitedTips：`{group,opUser,invitedUserList[],operationTime,groupMemberVersion,groupMemberVersionID,inviterUser}`

**1510 成员主动入群** — MemberEnterTips：`{group,entrantUser,operationTime,groupMemberVersion,groupMemberVersionID}`

**1511 群组已解散** — GroupDismissedTips：`{group,opUser,operationTime}`

**1512 群成员被禁言** — GroupMemberMutedTips：

```json
{"group":{"...":"GroupInfo"},"opUser":{"...":"..."},"operationTime":1650000000000,"mutedUser":{"...":"..."},"mutedSeconds":600,"groupMemberVersion":9,"groupMemberVersionID":"..."}
```

**1513 群成员取消禁言** — GroupMemberCancelMutedTips：同 1512，无 `mutedSeconds`。

**1514 群组全员禁言 / 1515 取消全员禁言** — GroupMutedTips / GroupCancelMutedTips：`{group,opUser,operationTime,groupMemberVersion,groupMemberVersionID}`

**1516 群成员信息已设置 / 1517 设为管理员 / 1518 设为普通成员** — GroupMemberInfoSetTips：`{group,opUser,operationTime,changedUser,groupMemberVersion,groupMemberVersionID,groupSortVersion}`

**1519 群公告已设置** — GroupInfoSetAnnouncementTips：`{opUser,group,groupMemberVersion,groupMemberVersionID}`

**1520 群名称已设置** — GroupInfoSetNameTips：`{opUser,group,groupMemberVersion,groupMemberVersionID}`

### 4.5 会话与消息通知

**1300 会话变更 / 1702 会话未读** — ConversationUpdateTips：

```json
{"userID":"user_001","conversationIDList":["si_user_001_user_002"]}
```

**1701 会话私有聊天开关** — ConversationSetPrivateTips：

```json
{"recvID":"user_002","sendID":"user_001","isPrivate":true,"conversationID":"si_user_001_user_002"}
```

**1703 清空会话** — ClearConversationTips：

```json
{"userID":"user_001","conversationIDs":["si_user_001_user_002"]}
```

**1704 删除会话** — ConversationDeleteTips：

```json
{"userID":"user_001","conversationIDs":["si_user_001_user_002"]}
```

**1652 消息删除通知（超级群）** — DeleteMessageTips：

```json
{"opUserID":"user_001","userID":"user_002","seqs":[101,102]}
```

**2101 消息撤回通知** — RevokeMsgTips：

```json
{
  "revokerUserID":"user_001",
  "clientMsgID":"client-uuid-0001",
  "revokeTime":1650000001000,
  "sesstionType":1,
  "seq":42,
  "conversationID":"si_user_001_user_002",
  "isAdminRevoke":false
}
```

**2102 删除消息通知** — DeleteMsgsTips：

```json
{"userID":"user_001","conversationID":"si_user_001_user_002","seqs":[41,42]}
```

**2200 已读回执** — MarkAsReadTips：

```json
{"markAsReadUserID":"user_002","conversationID":"si_user_001_user_002","seqs":[40,41,42],"hasReadSeq":42}
```

单聊中接收方标记已读后，发送方各端收到此通知以更新"已读"状态。

---

## 5. 消息选项 options

`options` 为键值开关，控制消息的存储/推送/计数行为：

| 键 | 说明 |
|---|---|
| `history` | 是否历史消息（拉取可见） |
| `persistent` | 是否持久化落库 |
| `offlinePush` | 是否离线推送（APNs/FCM 等） |
| `unreadCount` | 是否计入未读数 |
| `conversationUpdate` | 是否更新接收端会话 |
| `senderSync` | 是否同步到发送者其他端 |
| `notPrivate` | 是否非私有（通知类消息置 true） |
| `senderConversationUpdate` | 是否更新发送端会话 |
| `senderNotificationPush` | 是否发送端通知推送 |
| `reactionFromCache` | 表情回应是否走缓存 |
| `isNotNotification` | 是否非通知（true 时为普通消息，会话 ID 使用 si_/g_/sg_ 前缀） |
| `isSendMsg` | 是否作为消息发送 |

典型组合：

| 场景 | history | persistent | offlinePush | unreadCount | conversationUpdate | senderSync |
|---|---|---|---|---|---|---|
| 普通聊天消息 | true | true | true | true | true | true |
| Typing | false | false | false | false | false | false |
| 撤回 | false | true | false | false | true | true |
| 仅在线自定义 | false | false | false | false | false | false |
| 系统通知（可靠） | true | true | true | false | true | false |

---

## 6. 场景完整示例

以下示例省略未使用字段，`options` 采用典型组合。

### 6.1 单聊文本消息

```json
{
  "sendID":"user_001","recvID":"user_002","groupID":"",
  "clientMsgID":"client-uuid-0001","senderPlatformID":1,
  "senderNickname":"Alice","senderFaceURL":"https://cdn.example.com/u/001.png",
  "sessionType":1,"msgFrom":100,"contentType":101,
  "content":"{\"text\":\"Hello, 世界\",\"atUsers\":[]}",
  "options":{"history":true,"persistent":true,"offlinePush":true,"unreadCount":true,"conversationUpdate":true,"senderSync":true,"isSendMsg":true},
  "offlinePushInfo":{"title":"Alice","desc":"Hello, 世界","iOSPushSound":"default","iOSBadgeCount":true}
}
```

### 6.2 群聊 @ 消息

```json
{
  "sendID":"user_001","recvID":"","groupID":"group_100",
  "clientMsgID":"client-uuid-0003","senderPlatformID":1,
  "senderNickname":"Alice","senderFaceURL":"https://cdn.example.com/u/001.png",
  "sessionType":3,"msgFrom":100,"contentType":106,
  "content":"{\"text\":\"@user_002 加油吧\",\"atUsers\":[{\"atUserID\":\"user_002\",\"subTop\":\"user_002\"}]}",
  "atUserIDList":["user_002"],
  "options":{"history":true,"persistent":true,"offlinePush":true,"unreadCount":true,"conversationUpdate":true,"senderSync":true,"isSendMsg":true},
  "offlinePushInfo":{"title":"项目组","desc":"Alice: @user_002 加油吧","iOSPushSound":"default","iOSBadgeCount":true}
}
```

### 6.3 图片消息（单聊）

```json
{
  "sendID":"user_001","recvID":"user_002",
  "clientMsgID":"client-uuid-0005","senderPlatformID":1,
  "sessionType":1,"msgFrom":100,"contentType":102,
  "content":"{\"origPictureElem\":{\"url\":\"https://cdn.example.com/img/p=1\",\"width\":1920,\"height\":1080,\"fileSize\":500000},\"bigPictureElem\":{\"url\":\"https://cdn.example.com/img/p=1-big\",\"width\":3840,\"height\":2160},\"snapShotElem\":{\"url\":\"https://cdn.example.com/img/p=1-thumb\",\"width\":300,\"height\":300}}",
  "options":{"history":true,"persistent":true,"offlinePush":false,"unreadCount":true,"conversationUpdate":true,"senderSync":true,"isSendMsg":true},
  "offlinePushInfo":{"title":"Alice","desc":"[图片]","iOSPushSound":"default","iOSBadgeCount":true}
}
```

### 6.4 正在输入（不落库、不推送）

```json
{
  "sendID":"user_001","recvID":"user_002",
  "clientMsgID":"client-uuid-0015","senderPlatformID":1,
  "sessionType":1,"msgFrom":100,"contentType":113,
  "content":"{\"msgContent\":\"1\"}",
  "options":{"history":false,"persistent":false,"offlinePush":false,"unreadCount":false,"conversationUpdate":false,"senderSync":false,"isNotNotification":true,"isSendMsg":false}
}
```

### 6.5 撤回消息

```json
{
  "sendID":"user_001","recvID":"user_002",
  "clientMsgID":"client-uuid-0016","senderPlatformID":1,
  "sessionType":1,"msgFrom":100,"contentType":111,
  "content":"{\"clientMsgID\":\"client-uuid-0001\",\"revokerID\":\"user_001\",\"revokerNickname\":\"Alice\",\"revokerRole\":0,\"sourceMessageSendID\":\"user_001\",\"sourceMessageSenderNickname\":\"Alice\",\"sessionType\":1,\"seq\":1}",
  "options":{"history":false,"persistent":true,"offlinePush":false,"unreadCount":false,"conversationUpdate":true,"senderSync":true,"isSendMsg":false}
}
```

### 6.6 系统通知（msgFrom=200）

```json
{
  "sendID":"system","recvID":"user_002",
  "clientMsgID":"client-uuid-0017","senderNickname":"系统助手",
  "sessionType":1,"msgFrom":200,"contentType":203,
  "content":"{\"serviceName\":\"platform_login\",\"data\":\"{\\\"service\\\":\\\"platform_login\\\",\\\"time\\\":1650000000000}\"}",
  "options":{"history":true,"persistent":true,"offlinePush":true,"unreadCount":false,"conversationUpdate":true,"notPrivate":true,"isNotNotification":true,"isSendMsg":true},
  "offlinePushInfo":{"title":"系统助手","desc":"平台登录提醒","iOSPushSound":"default","iOSBadgeCount":true}
}
```

### 6.7 群创建通知（1501）

```json
{
  "sendID":"user_001","recvID":"user_002",
  "sessionType":4,"msgFrom":200,"contentType":1501,
  "content":"{\"detail\":\"{\\\"group\\\":{\\\"groupID\\\":\\\"group_100\\\",\\\"groupName\\\":\\\"项目组\\\",...},\\\"opUser\\\":{\\\"userID\\\":\\\"user_001\\\",\\\"roleLevel\\\":100,...},\\\"memberList\\\":[...],\\\"operationTime\\\":1650000000000}\"}",
  "options":{"history":true,"persistent":true,"unreadCount":false,"conversationUpdate":true}
}
```

> 通知的 `content` 为 NotificationElem 封装：`{"detail":"<Tips JSON 字符串>"}`，`detail` 需二次解析。

### 6.8 好友申请通知（1203）

```json
{
  "sendID":"user_001","recvID":"user_002",
  "sessionType":4,"msgFrom":200,"contentType":1203,
  "content":"{\"detail\":\"{\\\"fromToUserID\\\":{\\\"fromUserID\\\":\\\"user_001\\\",\\\"toUserID\\\":\\\"user_002\\\"},\\\"request\\\":{\\\"fromUserID\\\":\\\"user_001\\\",\\\"reqMsg\\\":\\\"交个朋友\\\",\\\"handleResult\\\":0,...}}\"}",
  "options":{"history":true,"persistent":true,"unreadCount":false,"conversationUpdate":true}
}
```

---

## 附录 A. 平台 ID（senderPlatformID）

`1`IOS / `2`Android / `3`Windows / `4`OSX / `5`Web / `6`MiniWeb / `7`Linux / `8`APad / `9`IPad / `10`Admin / `11`HarmonyOS / `12`Bot。

## 附录 B. 其他常用枚举

- **群类型 groupType**：`0`普通群 / `1`超级群 / `2`工作群
- **群状态 status**：`0`正常 / `1`禁止发言 / `2`已解散 / `3`已禁言 / `4`禁止私聊
- **群角色 roleLevel**：`100`群主 / `60`管理员 / `20`普通成员
- **入群方式 joinSource**：`1`管理员添加 / `2`邀请 / `3`搜索 / `4`扫码
- **加群验证 needVerification**：`0`申请需验证邀请直接进 / `1`所有人都需验证 / `2`直接加入
- **申请处理结果 handleResult**：`0`未处理 / `1`已同意 / `-1`已拒绝
- **全局消息接收 globalRecvMsgOpt**：`0`接收 / `1`不接收 / `2`接收不通知
- **@类型**：`0`普通@ / `1`@我 / `2`@所有人 / `3`@所有人且@我 / `4`群公告
