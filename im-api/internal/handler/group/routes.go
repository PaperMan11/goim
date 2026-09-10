package group

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

func InitRoutes(r *gin.RouterGroup, svc *svc.ServiceContext) {
	groupRoute := r.Group("/group")

	groupRoute.POST("/create_group", CreateGroup(svc))
	groupRoute.POST("/set_group_info", SetGroupInfo(svc))
	groupRoute.POST("/set_group_info_ex", SetGroupInfoEx(svc))
	groupRoute.POST("/join_group", JoinGroup(svc))
	groupRoute.POST("/quit_group", QuitGroup(svc))
	groupRoute.POST("/group_application_response", GroupApplicationResponse(svc))
	groupRoute.POST("/transfer_group", TransferGroup(svc))
	groupRoute.POST("/get_recv_group_applicationList", GetRecvGroupApplicationList(svc))
	groupRoute.POST("/get_user_req_group_applicationList", GetUserReqGroupApplicationList(svc))
	groupRoute.POST("/get_group_users_req_application_list", GetGroupUsersReqApplicationList(svc))
	groupRoute.POST("/get_specified_user_group_request_info", GetSpecifiedUserGroupRequestInfo(svc))
	groupRoute.POST("/get_groups_info", GetGroupsInfo(svc))
	groupRoute.POST("/kick_group", KickGroup(svc))
	groupRoute.POST("/get_group_members_info", GetGroupMembersInfo(svc))
	groupRoute.POST("/get_group_member_list", GetGroupMemberList(svc))
	groupRoute.POST("/invite_user_to_group", InviteUserToGroup(svc))
	groupRoute.POST("/get_joined_group_list", GetJoinedGroupList(svc))
	groupRoute.POST("/dismiss_group", DismissGroup(svc))
	groupRoute.POST("/mute_group_member", MuteGroupMember(svc))
	groupRoute.POST("/cancel_mute_group_member", CancelMuteGroupMember(svc))
	groupRoute.POST("/mute_group", MuteGroup(svc))
	groupRoute.POST("/cancel_mute_group", CancelMuteGroup(svc))
	groupRoute.POST("/set_group_member_info", SetGroupMemberInfo(svc))
	groupRoute.POST("/get_group_abstract_info", GetGroupAbstractInfo(svc))
	groupRoute.POST("/get_groups", GetGroups(svc))
	groupRoute.POST("/get_group_member_user_id", GetGroupMemberUserID(svc))
	groupRoute.POST("/get_incremental_join_groups", GetIncrementalJoinGroups(svc))
	groupRoute.POST("/get_incremental_group_members", GetIncrementalGroupMembers(svc))
	groupRoute.POST("/get_incremental_group_members_batch", GetIncrementalGroupMembersBatch(svc))
	groupRoute.POST("/get_full_group_member_user_ids", GetFullGroupMemberUserIDs(svc))
	groupRoute.POST("/get_full_join_group_ids", GetFullJoinGroupIDs(svc))
	groupRoute.POST("/get_group_application_unhandled_count", GetGroupApplicationUnhandledCount(svc))
}
