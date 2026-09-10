package relation

import (
	"github.com/PaperMan11/goim/im-api/internal/svc"
	"github.com/gin-gonic/gin"
)

func InitRoutes(r *gin.RouterGroup, svc *svc.ServiceContext) {
	relationRoute := r.Group("/relation")

	relationRoute.POST("/delete_friend", DeleteFriend(svc))
	relationRoute.POST("/get_friend_apply_list", GetFriendApplyList(svc))
	relationRoute.POST("/get_designated_friend_apply", GetDesignatedFriendApply(svc))
	relationRoute.POST("/get_self_friend_apply_list", GetSelfFriendApplyList(svc))
	relationRoute.POST("/get_friend_list", GetFriendList(svc))
	relationRoute.POST("/get_designated_friends", GetDesignatedFriends(svc))
	relationRoute.POST("/add_friend", AddFriend(svc))
	relationRoute.POST("/add_friend_response", AddFriendResponse(svc))
	relationRoute.POST("/set_friend_remark", SetFriendRemark(svc))
	relationRoute.POST("/add_black", AddBlack(svc))
	relationRoute.POST("/get_black_list", GetBlackList(svc))
	relationRoute.POST("/get_specified_blacks", GetSpecifiedBlacks(svc))
	relationRoute.POST("/remove_black", RemoveBlack(svc))
	relationRoute.POST("/get_incremental_blacks", GetIncrementalBlacks(svc))
	relationRoute.POST("/import_friend", ImportFriend(svc))
	relationRoute.POST("/is_friend", IsFriend(svc))
	relationRoute.POST("/get_friend_id", GetFriendID(svc))
	relationRoute.POST("/get_specified_friends_info", GetSpecifiedFriendsInfo(svc))
	relationRoute.POST("/update_friends", UpdateFriends(svc))
	relationRoute.POST("/get_incremental_friends", GetIncrementalFriends(svc))
	relationRoute.POST("/get_full_friend_user_ids", GetFullFriendUserIDs(svc))
	relationRoute.POST("/get_self_unhandled_apply_count", GetSelfUnhandledApplyCount(svc))
}
