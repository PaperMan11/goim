package msgdispatcher

type OfflinePushConfig struct {
	Enable bool
	Title  string
	Desc   string
	Ext    string
}

type NotificationConfig struct {
	IsSendMsg        bool
	ReliabilityLevel int
	UnreadCount      bool
	OfflinePush      OfflinePushConfig
}

type Notification struct {
	GroupCreated              NotificationConfig
	GroupInfoSet              NotificationConfig
	JoinGroupApplication      NotificationConfig
	MemberQuit                NotificationConfig
	GroupApplicationAccepted  NotificationConfig
	GroupApplicationRejected  NotificationConfig
	GroupOwnerTransferred     NotificationConfig
	MemberKicked              NotificationConfig
	MemberInvited             NotificationConfig
	MemberEnter               NotificationConfig
	GroupDismissed            NotificationConfig
	GroupMuted                NotificationConfig
	GroupCancelMuted          NotificationConfig
	GroupMemberMuted          NotificationConfig
	GroupMemberCancelMuted    NotificationConfig
	GroupMemberInfoSet        NotificationConfig
	GroupMemberSetToAdmin     NotificationConfig
	GroupMemberSetToOrdinary  NotificationConfig
	GroupInfoSetAnnouncement  NotificationConfig
	GroupInfoSetName          NotificationConfig
	FriendApplicationAdded    NotificationConfig
	FriendApplicationApproved NotificationConfig
	FriendApplicationRejected NotificationConfig
	FriendAdded               NotificationConfig
	FriendDeleted             NotificationConfig
	FriendRemarkSet           NotificationConfig
	BlackAdded                NotificationConfig
	BlackDeleted              NotificationConfig
	FriendInfoUpdated         NotificationConfig
	UserInfoUpdated           NotificationConfig
	UserStatusChanged         NotificationConfig
	ConversationChanged       NotificationConfig
	ConversationSetPrivate    NotificationConfig
}
