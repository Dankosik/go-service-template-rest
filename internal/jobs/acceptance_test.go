package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestJobsAcceptance(t *testing.T) {
	t.Parallel()
	t.Run("identity bounds", func(t *testing.T) {
		t.Parallel()
		setters := []struct {
			name string
			set  func(*AcceptanceIdentity, string)
		}{
			{name: "logical", set: func(i *AcceptanceIdentity, v string) { i.LogicalJobID = LogicalJobID(v) }},
			{name: "producer scope", set: func(i *AcceptanceIdentity, v string) { i.ProducerScope = ProducerScope(v) }},
			{name: "producer key", set: func(i *AcceptanceIdentity, v string) { i.ProducerKey = ProducerKey(v) }},
			{name: "occurrence scope", set: func(i *AcceptanceIdentity, v string) { i.OccurrenceScope = OccurrenceScope(v) }},
			{name: "occurrence id", set: func(i *AcceptanceIdentity, v string) { i.OccurrenceID = OccurrenceID(v) }},
			{name: "effect scope", set: func(i *AcceptanceIdentity, v string) { i.EffectScope = EffectScope(v) }},
			{name: "effect key", set: func(i *AcceptanceIdentity, v string) { i.EffectKey = EffectKey(v) }},
		}
		for _, setter := range setters {
			for _, size := range []int{0, 1, MaxIdentityBytes, MaxIdentityBytes + 1} {
				identity := testIdentity()
				setter.set(&identity, strings.Repeat("x", size))
				err := identity.Validate()
				wantValid := size == 1 || size == MaxIdentityBytes
				if wantValid && err != nil {
					t.Fatalf("%s size %d error = %v", setter.name, size, err)
				}
				if !wantValid && !errors.Is(err, ErrInvalidAcceptance) {
					t.Fatalf("%s size %d error = %v, want ErrInvalidAcceptance", setter.name, size, err)
				}
			}
		}
	})

	definition := testDefinition(t, Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"})
	first, err := definition.Prepare(testArgs{Task: "send", Count: 1, Metadata: map[string]string{"b": "2", "a": "1"}}, testIdentity(), testAvailableAt())
	requireError(t, err)
	second, err := definition.Prepare(testArgs{Task: "send", Count: 1, Metadata: map[string]string{"a": "1", "b": "2"}}, testIdentity(), testAvailableAt())
	requireError(t, err)
	if first.Fingerprint() != second.Fingerprint() {
		t.Fatal("equivalent typed intent produced different fingerprints")
	}
	expectation := first.ReadbackExpectation()
	if err := expectation.Validate(); err != nil {
		t.Fatalf("ReadbackExpectation.Validate() error = %v", err)
	}
	if expectation.Identity() != first.Identity() || expectation.Fingerprint() != first.Fingerprint() {
		t.Fatal("readback expectation did not preserve immutable intent")
	}
	if err := (ReadbackExpectation{}).Validate(); !errors.Is(err, ErrInvalidAcceptance) {
		t.Fatalf("zero ReadbackExpectation.Validate() error = %v", err)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("Prepared.Validate() error = %v", err)
	}
	if err := (Prepared{}).Validate(); !errors.Is(err, ErrInvalidAcceptance) {
		t.Fatalf("zero Prepared.Validate() error = %v", err)
	}
	const wantFingerprint = "010d0dcf13d95b61368e0d6a003c6d9878a92817d5e8325834e886bee8985f0e"
	if got := fmt.Sprintf("%x", first.Fingerprint()); got != wantFingerprint {
		t.Fatalf("fingerprint = %s, want %s", got, wantFingerprint)
	}
	changed, err := definition.Prepare(testArgs{Task: "send", Count: 2, Metadata: map[string]string{"a": "1", "b": "2"}}, testIdentity(), testAvailableAt())
	requireError(t, err)
	if first.Fingerprint() == changed.Fingerprint() {
		t.Fatal("changed semantic field retained fingerprint")
	}
	if first.ReadbackExpectation().Fingerprint() == changed.ReadbackExpectation().Fingerprint() {
		t.Fatal("changed semantic field retained readback fingerprint")
	}

	t.Run("JSON object order is not intent", func(t *testing.T) {
		t.Parallel()
		type rawArgs struct {
			Task string          `json:"task"`
			Data json.RawMessage `json:"data"`
		}
		input := DefinitionInput[rawArgs]{
			Revision:        Revision{Kind: "raw", ArgsVersion: "v1", PolicyVersion: "p1"},
			MaxPayloadBytes: 1024,
			Validate: func(args rawArgs) error {
				if args.Task == "" || len(args.Data) == 0 {
					return errors.New("required")
				}
				return nil
			},
			Policy: testPolicy(),
		}
		rawDefinition, err := NewDefinition(input)
		requireError(t, err)
		left, err := rawDefinition.Prepare(rawArgs{Task: "send", Data: json.RawMessage(`{"a":1,"b":2}`)}, testIdentity(), testAvailableAt())
		requireError(t, err)
		right, err := rawDefinition.Prepare(rawArgs{Task: "send", Data: json.RawMessage(`{"b":2,"a":1}`)}, testIdentity(), testAvailableAt())
		requireError(t, err)
		if left.Fingerprint() != right.Fingerprint() {
			t.Fatal("equivalent JSON object order changed fingerprint")
		}
	})

	payload := first.Payload()
	payload[0] ^= 0xff
	if first.Payload()[0] == payload[0] {
		t.Fatal("Prepared.Payload aliases internal bytes")
	}

	t.Run("closed results", func(t *testing.T) {
		t.Parallel()
		stages := []StageResult{
			{Outcome: StageNew, LogicalJobID: "job-1"},
			{Outcome: StageExisting, LogicalJobID: "job-1"},
			{Outcome: StageConflict, LogicalJobID: "job-1"},
			{Outcome: StageRejected},
		}
		for _, result := range stages {
			if err := result.Validate(); err != nil {
				t.Fatalf("StageResult(%q).Validate() error = %v", result.Outcome, err)
			}
		}
		stageOutcomes := map[StageOutcome]struct{}{}
		for _, result := range stages {
			stageOutcomes[result.Outcome] = struct{}{}
		}
		if len(stageOutcomes) != len(stages) {
			t.Fatal("known stage outcomes are not pairwise distinct")
		}
		if err := (StageResult{Outcome: "future"}).Validate(); !errors.Is(err, ErrInvalidAcceptance) {
			t.Fatalf("unknown StageResult error = %v", err)
		}

		readbacks := []ReadbackResult{
			{Outcome: ReadbackAccepted, LogicalJobID: "job-1"},
			{Outcome: ReadbackNotAccepted},
			{Outcome: ReadbackConflict, LogicalJobID: "job-1"},
			{Outcome: ReadbackUnknown},
		}
		for _, result := range readbacks {
			if err := result.Validate(); err != nil {
				t.Fatalf("ReadbackResult(%q).Validate() error = %v", result.Outcome, err)
			}
		}
		readbackOutcomes := map[ReadbackOutcome]struct{}{}
		for _, result := range readbacks {
			readbackOutcomes[result.Outcome] = struct{}{}
		}
		if len(readbackOutcomes) != len(readbacks) {
			t.Fatal("known readback outcomes are not pairwise distinct")
		}
		if err := (ReadbackResult{Outcome: "future"}).Validate(); !errors.Is(err, ErrInvalidAcceptance) {
			t.Fatalf("unknown ReadbackResult error = %v", err)
		}
	})
}
