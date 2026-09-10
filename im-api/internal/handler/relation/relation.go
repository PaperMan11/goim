package relation

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/PaperMan11/goim/pkg/a2r"
	"github.com/gin-gonic/gin"
)

func DeleteFriend(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.DeleteFriend)
	}
}

func GetFriendApplyList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetPaginationFriendsApplyTo)
	}
}

func GetDesignatedFriendApply(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetDesignatedFriendsApply)
	}
}

func GetSelfFriendApplyList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetPaginationFriendsApplyFrom)
	}
}

func GetFriendList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetPaginationFriends)
	}
}

func GetDesignatedFriends(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetDesignatedFriends)
	}
}

func AddFriend(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.ApplyToAddFriend)
	}
}

func AddFriendResponse(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.RespondFriendApply)
	}
}

func SetFriendRemark(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.SetFriendRemark)
	}
}

func AddBlack(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.AddBlack)
	}
}

func GetBlackList(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetPaginationBlacks)
	}
}

func GetSpecifiedBlacks(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetSpecifiedBlacks)
	}
}

func RemoveBlack(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.RemoveBlack)
	}
}

func GetIncrementalBlacks(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetIncrementalBlacks)
	}
}

func ImportFriend(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.ImportFriends)
	}
}

func IsFriend(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.IsFriend)
	}
}

func GetFriendID(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetFriendIDs)
	}
}

func GetSpecifiedFriendsInfo(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetSpecifiedFriendsInfo)
	}
}

func UpdateFriends(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.UpdateFriends)
	}
}

func GetIncrementalFriends(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetIncrementalFriends)
	}
}

func GetFullFriendUserIDs(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetFullFriendUserIDs)
	}
}

func GetSelfUnhandledApplyCount(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		a2r.Call(c, svc.RelationService.GetSelfUnhandledApplyCount)
	}
}
