//go:build jobs_test_worker

package bootstrap

import "os"

func recordUnsafeDrain() {
	writeLifecycleMarker(os.Getenv("JOBS_WORKER_TEST_UNSAFE_DRAIN_FILE"), "unsafe drain\n")
}

func recordProhibitedUnsafeCleanup() {
	writeLifecycleMarker(os.Getenv("JOBS_WORKER_TEST_UNSAFE_CLEANUP_FILE"), "unsafe cleanup\n")
}

func writeLifecycleMarker(path, value string) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(value), 0o600)
}
