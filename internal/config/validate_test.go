package config

import (
	"errors"
	"testing"
)

func TestHealthRefreshBounds(t *testing.T) {
	for _, tc := range []struct {
		name      string
		interval  string
		threshold string
		wantErr   bool
	}{
		{name: "interval too small", interval: "50ms", wantErr: true},
		{name: "interval too large", interval: "2m", wantErr: true},
		{name: "threshold zero", threshold: "0", wantErr: true},
		{name: "threshold too large", threshold: "101", wantErr: true},
		{name: "threshold one accepted", threshold: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			if tc.interval != "" {
				t.Setenv("APP__HEALTH__REFRESH_INTERVAL", tc.interval)
			}
			if tc.threshold != "" {
				t.Setenv("APP__HEALTH__FAILURE_THRESHOLD", tc.threshold)
			}

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr && !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

func TestRuntimeMemoryLimitRatioBounds(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ratio   string
		wantErr bool
	}{
		{name: "zero disables detection", ratio: "0"},
		{name: "one accepted", ratio: "1"},
		{name: "negative", ratio: "-0.1", wantErr: true},
		{name: "above one", ratio: "1.1", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			if tc.ratio != "" {
				t.Setenv("APP__RUNTIME__MEMORY_LIMIT_RATIO", tc.ratio)
			}

			_, _, err := LoadDetailed(LoadOptions{})
			if tc.wantErr && err == nil {
				t.Fatal("LoadDetailed() error = nil, want non-nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}
