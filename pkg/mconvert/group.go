package mconvert

import (
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
)

func ModelToPbGroupInfo(group *model.Group) *sdkws.GroupInfo {
	if group == nil {
		return nil
	}
	return &sdkws.GroupInfo{
		GroupID:                group.GroupID,
		GroupName:              group.GroupName,
		Notification:           group.Notification,
		Introduction:           group.Introduction,
		OwnerUserID:            group.OwnerUserID,
		FaceURL:                group.FaceURL,
		CreateTime:             group.CreateTime.UnixMilli(),
		MemberCount:            uint32(group.MemberCount),
		Ex:                     group.Extra,
		Status:                 int32(group.Status),
		CreatorUserID:          group.CreatorUserID,
		GroupType:              int32(group.GroupType),
		NeedVerification:       int32(group.NeedVerification),
		LookMemberInfo:         int32(group.LookMemberInfo),
		ApplyMemberFriend:      int32(group.ApplyMemberFriend),
		NotificationUpdateTime: group.NotificationUpdateTime.UnixMilli(),
		NotificationUserID:     group.NotificationUserID,
	}
}

func ModelToPbGroupMemberInfo(member *model.GroupMember) *sdkws.GroupMemberFullInfo {
	if member == nil {
		return nil
	}
	return &sdkws.GroupMemberFullInfo{
		GroupID:        member.GroupID,
		UserID:         member.UserID,
		RoleLevel:      int32(member.RoleLevel),
		JoinTime:       member.JoinTime.UnixMilli(),
		Nickname:       member.Nickname,
		FaceURL:        member.FaceURL,
		AppMangerLevel: int32(member.AppManagerLevel),
		JoinSource:     int32(member.JoinSource),
		OperatorUserID: member.OperatorUserID,
		Ex:             member.Extra,
		MuteEndTime:    member.MuteEndTime.UnixMilli(),
		InviterUserID:  member.InviterUserID,
	}
}

func ModelToPbGroupRequest(req *model.GroupRequest, user *sdkws.UserInfo, group *sdkws.GroupInfo) *sdkws.GroupRequest {
	if req == nil {
		return nil
	}

	var pubUser *sdkws.PublicUserInfo
	if user != nil {
		pubUser = &sdkws.PublicUserInfo{
			UserID:   user.UserID,
			Nickname: user.Nickname,
			FaceURL:  user.FaceURL,
			Ex:       user.Ex,
		}
	} else {
		pubUser = &sdkws.PublicUserInfo{
			UserID:   req.UserID,
			Nickname: req.Nickname,
			FaceURL:  req.FaceURL,
		}
	}

	if group == nil {
		group = &sdkws.GroupInfo{
			GroupID:   req.GroupID,
			GroupName: req.GroupName,
			FaceURL:   req.GroupFaceURL,
		}
	}
	return &sdkws.GroupRequest{
		UserInfo:      pubUser,
		GroupInfo:     group,
		HandleResult:  int32(req.HandleResult),
		ReqMsg:        req.ReqMsg,
		HandleMsg:     req.HandleMsg,
		ReqTime:       req.ReqTime.UnixMilli(),
		HandleUserID:  req.HandleUserID,
		HandleTime:    req.HandleTime.UnixMilli(),
		Ex:            req.Extra,
		JoinSource:    int32(req.JoinSource),
		InviterUserID: req.InviterUserID,
	}
}
