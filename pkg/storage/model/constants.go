package model

const (
	CollectionUser             = "im_users"
	CollectionUserStatus       = "im_user_status"
	CollectionUserCommand      = "im_user_commands"
	CollectionUserClientConfig = "im_user_client_configs"

	CollectionGroup       = "im_groups"
	CollectionGroupMember = "im_group_members"

	CollectionMessage         = "im_messages"
	CollectionSeqConversation = "im_seq_conversations"
	CollectionSeqUser         = "im_seq_users"

	CollectionConversation          = "im_conversations"
	CollectionConversationLatestMsg = "im_conversation_latest_msgs"

	CollectionFriend = "im_friends"
	CollectionBlack  = "im_blacks"

	CollectionFriendRequest = "im_friend_requests"
	CollectionGroupRequest  = "im_group_requests"

	CollectionWebhookDelivery = "im_webhook_deliveries"

	// version
	CollectionGroupVersion        = "im_group_versions"
	CollectionFriendVersion       = "im_friend_versions"
	CollectionConversationVersion = "im_conversation_versions"
	CollectionBlackVersion        = "im_black_versions"
	CollectionUserVersion         = "im_user_versions"
)
