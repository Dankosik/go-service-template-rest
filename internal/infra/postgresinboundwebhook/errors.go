// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import "errors"

var (
	errDecoderFailed      = errors.New("inbound webhook decoder failed")
	errHandlerFailed      = errors.New("inbound webhook handler failed")
	errStorageUnavailable = errors.New("inbound webhook storage unavailable")
	errPanicRecovered     = errors.New("inbound webhook panic recovered")
)

const (
	logClassDecoderInternal          = "decoder_internal"
	logClassHandlerRetryable         = "handler_retryable"
	logClassStorageRetryable         = "storage_retryable"
	logClassBindingUnavailable       = "binding_unavailable"
	logClassPanicRecovered           = "panic_recovered"
	logClassTerminalizationRetryable = "terminalization_retryable"
)

// profile:inbound-webhooks-standard:end
