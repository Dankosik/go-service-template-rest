package config

import (
	"errors"
	"testing"
)

func TestGRPCEnabledTLSValidation(t *testing.T) {
	t.Parallel()
	valid := GRPCConfig{Server: GRPCServerConfig{
		Enabled:           true,
		Addr:              " :9091 ",
		TransportSecurity: " TLS ",
		TLS: GRPCTLSConfig{
			CertFile:     " /run/secrets/service.crt ",
			KeyFile:      " /run/secrets/service.key ",
			ClientCAFile: " /run/secrets/clients.pem ",
		},
	}}
	if err := validateGRPCConfig(&valid); err != nil {
		t.Fatalf("validateGRPCConfig() error = %v", err)
	}
	if valid.Server.Addr != ":9091" || valid.Server.TransportSecurity != "tls" ||
		valid.Server.TLS.CertFile != "/run/secrets/service.crt" {
		t.Fatalf("gRPC config was not normalized: %+v", valid.Server)
	}

	for _, testCase := range []struct {
		name   string
		mutate func(*GRPCServerConfig)
	}{
		{name: "address", mutate: func(cfg *GRPCServerConfig) { cfg.Addr = "" }},
		{name: "mode", mutate: func(cfg *GRPCServerConfig) { cfg.TransportSecurity = "auto" }},
		{name: "certificate", mutate: func(cfg *GRPCServerConfig) { cfg.TLS.CertFile = "" }},
		{name: "key", mutate: func(cfg *GRPCServerConfig) { cfg.TLS.KeyFile = "" }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := valid
			testCase.mutate(&cfg.Server)
			if err := validateGRPCConfig(&cfg); !errors.Is(err, ErrValidate) {
				t.Fatalf("validateGRPCConfig() error = %v, want ErrValidate", err)
			}
		})
	}
}
