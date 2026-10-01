package messages

import (
	"context"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/registry"
	"github.com/clawscli/claws/internal/render"
)

func init() {
	registry.Global.RegisterCustom("sqs", "messages", registry.Entry{
		DAOFactory: func(ctx context.Context) (dao.DAO, error) {
			return NewMessageDAO(ctx)
		},
		RendererFactory: func() render.Renderer {
			return NewMessageRenderer()
		},
	})
}
