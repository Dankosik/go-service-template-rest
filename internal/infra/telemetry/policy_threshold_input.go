package telemetry

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	regexHealthcheckTimeout      = regexp.MustCompile(`(?m)^healthcheckTimeout\s*=\s*(\d+)\s*$`)
	regexRestartPolicyMaxRetries = regexp.MustCompile(`(?m)^restartPolicyMaxRetries\s*=\s*(\d+)\s*$`)
	regexOverlapSeconds          = regexp.MustCompile(`(?m)^overlapSeconds\s*=\s*(\d+)\s*$`)
	regexDrainingSeconds         = regexp.MustCompile(`(?m)^drainingSeconds\s*=\s*(\d+)\s*$`)
	regexReplicaBaseline         = regexp.MustCompile(`(?m)^#\s*-\s*production replica baseline:\s*>=\s*(\d+)\s*$`)
	regexPerReplicaBaseline      = regexp.MustCompile(`(?m)^#\s*-\s*per-replica baseline:\s*(\d+)\s*vCPU\s*/\s*(\d+)\s*GiB\s*$`)
)

// PolicyThresholdInputFromRailwayTOML extracts capacity and numeric policy thresholds
// from railway.toml source-of-truth content for release-readiness assertions.
func PolicyThresholdInputFromRailwayTOML(raw string) (PolicyThresholdInput, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return PolicyThresholdInput{}, fmt.Errorf("railway.toml content is empty")
	}

	healthcheckTimeoutSec, err := extractInt(regexHealthcheckTimeout, content, "healthcheckTimeout")
	if err != nil {
		return PolicyThresholdInput{}, err
	}
	restartMaxRetries, err := extractInt(regexRestartPolicyMaxRetries, content, "restartPolicyMaxRetries")
	if err != nil {
		return PolicyThresholdInput{}, err
	}
	overlapSeconds, err := extractInt(regexOverlapSeconds, content, "overlapSeconds")
	if err != nil {
		return PolicyThresholdInput{}, err
	}
	drainingSeconds, err := extractInt(regexDrainingSeconds, content, "drainingSeconds")
	if err != nil {
		return PolicyThresholdInput{}, err
	}
	replicaFloor, err := extractInt(regexReplicaBaseline, content, "production replica baseline")
	if err != nil {
		return PolicyThresholdInput{}, err
	}
	replicaCPU, replicaMemory, err := extractReplicaCaps(content)
	if err != nil {
		return PolicyThresholdInput{}, err
	}

	return PolicyThresholdInput{
		DeployHealthSLOTargetPercent: expectedDeployHealthSLOTargetPercent,
		CapacityReplicaFloor:         int64(replicaFloor),
		CapacityPerReplicaCPU:        int64(replicaCPU),
		CapacityPerReplicaMemoryGiB:  int64(replicaMemory),
		PromotionTimeout:             time.Duration(healthcheckTimeoutSec) * time.Second,
		DrainWindow:                  time.Duration(overlapSeconds) * time.Second,
		ShutdownTimeout:              time.Duration(drainingSeconds) * time.Second,
		RestartMaxRetries:            int64(restartMaxRetries),
	}, nil
}

func extractInt(regex *regexp.Regexp, content, fieldName string) (int, error) {
	match := regex.FindStringSubmatch(content)
	if len(match) != 2 {
		return 0, fmt.Errorf("railway.toml missing %s", fieldName)
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("railway.toml %s parse failed: %w", fieldName, err)
	}
	return value, nil
}

func extractReplicaCaps(content string) (int, int, error) {
	match := regexPerReplicaBaseline.FindStringSubmatch(content)
	if len(match) != 3 {
		return 0, 0, fmt.Errorf("railway.toml missing per-replica baseline comment")
	}

	cpu, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, fmt.Errorf("railway.toml per-replica cpu parse failed: %w", err)
	}
	memory, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, fmt.Errorf("railway.toml per-replica memory parse failed: %w", err)
	}

	return cpu, memory, nil
}
