// profile:inbound-webhooks-standard:start
package bootstrap

import (
	"context"
	"errors"

	"github.com/example/go-service-template-rest/internal/openapi"
)

var errInboundWebhookStrictFallback = errors.New("inbound webhook strict fallback is unreachable")

type serviceAPI struct {
	openapi.StrictServerInterface
}

func newServiceAPI() serviceAPI {
	return serviceAPI{}
}

func (serviceAPI) ReceiveWebhook(context.Context, openapi.ReceiveWebhookRequestObject) (openapi.ReceiveWebhookResponseObject, error) {
	return nil, errInboundWebhookStrictFallback
}

// profile:inbound-webhooks-standard:end
