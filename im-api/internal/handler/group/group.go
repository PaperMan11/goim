package group

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/a2r"
	"github.com/gin-gonic/gin"
)

func CreateGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.CreateGroup)
	}
}

func SetGroupInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.SetGroupInfo)
	}
}

func SetGroupInfoEx(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.SetGroupInfoEx)
	}
}

func JoinGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.JoinGroup)
	}
}

func QuitGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.QuitGroup)
	}
}

func GroupApplicationResponse(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GroupApplicationResponse)
	}
}

func TransferGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.TransferGroupOwner)
	}
}

func GetRecvGroupApplicationList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupApplicationList)
	}
}

func GetUserReqGroupApplicationList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetUserReqApplicationList)
	}
}

func GetGroupUsersReqApplicationList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupUsersReqApplicationList)
	}
}

func GetSpecifiedUserGroupRequestInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetSpecifiedUserGroupRequestInfo)
	}
}

func GetGroupsInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupsInfo)
	}
}

func KickGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.KickGroupMember)
	}
}

func GetGroupMembersInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupMembersInfo)
	}
}

func GetGroupMemberList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupMemberList)
	}
}

func InviteUserToGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.InviteUserToGroup)
	}
}

func GetJoinedGroupList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetJoinedGroupList)
	}
}

func DismissGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.DismissGroup)
	}
}

func MuteGroupMember(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.MuteGroupMember)
	}
}

func CancelMuteGroupMember(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.CancelMuteGroupMember)
	}
}

func MuteGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.MuteGroup)
	}
}

func CancelMuteGroup(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.CancelMuteGroup)
	}
}

func SetGroupMemberInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.SetGroupMemberInfo)
	}
}

func GetGroupAbstractInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupAbstractInfo)
	}
}

func GetGroups(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroups)
	}
}

func GetGroupMemberUserID(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupMemberUserIDs)
	}
}

func GetIncrementalJoinGroups(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetIncrementalJoinGroup)
	}
}

func GetIncrementalGroupMembers(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetIncrementalGroupMember)
	}
}

func GetIncrementalGroupMembersBatch(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.BatchGetIncrementalGroupMember)
	}
}

func GetFullGroupMemberUserIDs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetFullGroupMemberUserIDs)
	}
}

func GetFullJoinGroupIDs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetFullJoinGroupIDs)
	}
}

func GetGroupApplicationUnhandledCount(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.GroupService.GetGroupApplicationUnhandledCount)
	}
}
