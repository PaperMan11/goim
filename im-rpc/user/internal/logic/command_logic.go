package logic

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

func (l *Logic) ProcessUserCommandAdd(ctx context.Context, req *pbuser.ProcessUserCommandAddReq) (*pbuser.ProcessUserCommandAddResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	value := ""
	if req.GetValue() != nil {
		value = req.GetValue().GetValue()
	}
	extra := ""
	if req.GetEx() != nil {
		extra = req.GetEx().GetValue()
	}

	cmd := &model.UserCommand{
		UserID:     req.GetUserID(),
		Type:       int(req.GetType()),
		UUID:       req.GetUuid(),
		Value:      value,
		Extra:      extra,
		CreateTime: timex.Now(),
		UpdatedAt:  timex.Now(),
	}

	err := l.svcCtx.UserModel.InsertUserCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pbuser.ProcessUserCommandAddResp{}, nil
}

func (l *Logic) ProcessUserCommandUpdate(ctx context.Context, req *pbuser.ProcessUserCommandUpdateReq) (*pbuser.ProcessUserCommandUpdateResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	extra := ""
	if req.GetEx() != nil {
		extra = req.GetEx().GetValue()
	}
	value := ""
	if req.GetValue() != nil {
		value = req.GetValue().GetValue()
	}

	err := l.svcCtx.UserModel.UpdateUserCommand(ctx, req.GetUserID(), req.GetUuid(), value, extra)
	if err != nil {
		l.Errorf("UpdateUserCommand err: %v", err)
		return nil, err
	}

	return &pbuser.ProcessUserCommandUpdateResp{}, nil
}

func (l *Logic) ProcessUserCommandDelete(ctx context.Context, req *pbuser.ProcessUserCommandDeleteReq) (*pbuser.ProcessUserCommandDeleteResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	err := l.svcCtx.UserModel.DeleteUserCommand(ctx, req.GetUserID(), req.GetUuid())
	if err != nil {
		l.Errorf("DeleteUserCommand err: %v", err)
		return nil, err
	}

	return &pbuser.ProcessUserCommandDeleteResp{}, nil
}

func (l *Logic) ProcessUserCommandGet(ctx context.Context, req *pbuser.ProcessUserCommandGetReq) (*pbuser.ProcessUserCommandGetResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	cmds, err := l.svcCtx.UserModel.GetUserCommands(ctx, req.GetUserID(), req.GetType())
	if err != nil {
		l.Errorf("GetUserCommands err: %v", err)
		return nil, err
	}

	commands := make([]*pbuser.CommandInfoResp, 0, len(cmds))
	for _, cmd := range cmds {
		commands = append(commands, &pbuser.CommandInfoResp{
			Type:       int32(cmd.Type),
			Uuid:       cmd.UUID,
			Value:      cmd.Value,
			Ex:         cmd.Extra,
			CreateTime: cmd.CreateTime.UnixMilli(),
		})
	}

	return &pbuser.ProcessUserCommandGetResp{}, nil
}

func (l *Logic) ProcessUserCommandGetAll(ctx context.Context, req *pbuser.ProcessUserCommandGetAllReq) (*pbuser.ProcessUserCommandGetAllResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	cmds, err := l.svcCtx.UserModel.GetAllUserCommands(ctx, req.GetUserID())
	if err != nil {
		l.Errorf("GetAllUserCommands err: %v", err)
		return nil, err
	}

	commands := make([]*pbuser.AllCommandInfoResp, 0, len(cmds))
	for _, cmd := range cmds {
		commands = append(commands, &pbuser.AllCommandInfoResp{
			Type:       int32(cmd.Type),
			Uuid:       cmd.UUID,
			Value:      cmd.Value,
			Ex:         cmd.Extra,
			CreateTime: cmd.CreateTime.UnixMilli(),
		})
	}

	return &pbuser.ProcessUserCommandGetAllResp{
		CommandResp: commands,
	}, nil
}
