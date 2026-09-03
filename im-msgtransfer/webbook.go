package immsgtransfer

import (
	"context"

	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/webhooks"
	"github.com/zeromicro/go-zero/core/logc"
)

func (mt *MsgTransfer) triggerMessageSavedEvent(ctx context.Context, msg *sdkws.MsgData) {
	if err := mt.webhookManager.Dispatch(&webhooks.WebhookEvent{
		EventType: webhooks.EventMessageSaved,
		Data: map[string]interface{}{
			"client_msg_id": msg.ClientMsgID,
			"server_msg_id": msg.ServerMsgID,
			"seq":           msg.Seq,
		},
	}); err != nil {
		logc.Errorf(ctx, "failed to dispatch webhook event, err: %v", err)
	}
}
