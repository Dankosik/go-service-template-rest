package grpcclient

// Config identifies one logical backend. grpc-go owns resolution, reconnects,
// load balancing defaults, and transparent retries; this package adds only the
// fixed safety and observability options in New.
type Config struct {
	Target string
}

// DefaultConfig returns the complete default configuration for target.
func DefaultConfig(target string) Config {
	return Config{Target: target}
}
