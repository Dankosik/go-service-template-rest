package main

import (
	"bytes"
	"go/format"
	"strings"
	"testing"
)

func TestRenderMicroleaseEventDTOsPreservesContractFields(t *testing.T) {
	t.Parallel()

	source := render("abcdef")
	formatted, err := format.Source(source)
	if err != nil {
		t.Fatalf("rendered DTO source does not format: %v", err)
	}
	if !bytes.Contains(formatted, []byte("// proto-sha256: abcdef")) {
		t.Fatal("rendered source missing proto digest header")
	}
	for _, field := range []string{
		"MicroleaseChildTerminalSubmitted",
		"TerminalBasisFingerprint",
		"SafeExecutionReference",
		"MicroleaseCheckpointReported",
		"AllocatedChildCapSumUSDAtoms",
		"MicroleaseCloseReported",
		"FinalLocalStateFingerprint",
		"MicroleaseAdmissionRejected",
		"ReasonClass",
	} {
		if !strings.Contains(string(formatted), field) {
			t.Fatalf("rendered source missing contract field/type %q", field)
		}
	}
}
