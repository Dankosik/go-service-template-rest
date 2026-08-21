package grpcx

import (
	"errors"
	"time"
)

// MaxConnections is the fixed listener ceiling used by bootstrap. It remains
// separate from per-connection HTTP/2 streams and process-wide RPC admission.
const MaxConnections = 4096

const (
	defaultMaxConcurrentRPCs       = 256
	defaultMaxConcurrentHealthRPCs = MaxConnections
	defaultMaxConcurrentStreams    = 100
	defaultMaxHeaderListBytes      = 16 << 10
	defaultMaxReceiveMessageBytes  = 4 << 20
	defaultMaxSendMessageBytes     = 4 << 20
	defaultUnaryTimeout            = 8 * time.Second
)

// serverConfig is deliberately private: these are safe template defaults, not
// deployment knobs. Tests vary them only to falsify the policy itself.
type serverConfig struct {
	maxConcurrentRPCs       int
	maxConcurrentHealthRPCs int
	maxConcurrentStreams    uint32
	maxHeaderListBytes      uint32
	maxReceiveMessageBytes  int
	maxSendMessageBytes     int
	unaryTimeout            time.Duration
}

func defaultServerConfig() serverConfig {
	return serverConfig{
		maxConcurrentRPCs:       defaultMaxConcurrentRPCs,
		maxConcurrentHealthRPCs: defaultMaxConcurrentHealthRPCs,
		maxConcurrentStreams:    defaultMaxConcurrentStreams,
		maxHeaderListBytes:      defaultMaxHeaderListBytes,
		maxReceiveMessageBytes:  defaultMaxReceiveMessageBytes,
		maxSendMessageBytes:     defaultMaxSendMessageBytes,
		unaryTimeout:            defaultUnaryTimeout,
	}
}

func validateConfig(cfg serverConfig) error {
	if cfg.maxConcurrentRPCs <= 0 || cfg.maxConcurrentHealthRPCs <= 0 {
		return errors.New("build gRPC server: admission limits must be positive")
	}
	if cfg.maxConcurrentStreams == 0 || cfg.maxHeaderListBytes == 0 {
		return errors.New("build gRPC server: stream and header limits must be positive")
	}
	if cfg.maxReceiveMessageBytes <= 0 || cfg.maxSendMessageBytes <= 0 {
		return errors.New("build gRPC server: message limits must be positive")
	}
	if cfg.unaryTimeout <= 0 {
		return errors.New("build gRPC server: unary timeout must be positive")
	}
	return nil
}
