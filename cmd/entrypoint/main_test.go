package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const execMainEnv = "ENTRYPOINT_TEST_EXEC_MAIN"

// TestMain lets a test re-execute this test binary as the real entrypoint with
// a chosen argv[0], so the rejection path is observed on real process streams.
func TestMain(m *testing.M) {
	if os.Getenv(execMainEnv) == "1" {
		main()
	}
	os.Exit(m.Run())
}

type fakeTable struct {
	table []role
	calls []string
	args  [][]string
}

// newFakeTable stands in for the selected roles. The names are not role names,
// so this file names no role that a generated profile could leave unselected.
func newFakeTable(names ...string) *fakeTable {
	fake := &fakeTable{}
	for _, name := range names {
		fake.table = append(fake.table, role{name: name, main: func(args []string) int {
			fake.calls = append(fake.calls, name)
			fake.args = append(fake.args, args)
			return 7
		}})
	}
	return fake
}

func TestRunSelectsRoleByFinalArgv0Element(t *testing.T) {
	for _, argv0 := range []string{"alpha", "/alpha", "/usr/local/bin/alpha", "./alpha", "../x/alpha"} {
		fake := newFakeTable("service", "alpha")
		var stderr bytes.Buffer
		status := run("", []string{argv0, "-config", "c.yaml"}, fake.table, &stderr)
		if status != 7 || !slices.Equal(fake.calls, []string{"alpha"}) {
			t.Fatalf("argv0 %q: status=%d calls=%v", argv0, status, fake.calls)
		}
		if !slices.Equal(fake.args[0], []string{"-config", "c.yaml"}) {
			t.Fatalf("argv0 %q: role args=%v", argv0, fake.args[0])
		}
		if stderr.Len() != 0 {
			t.Fatalf("argv0 %q: stderr=%q", argv0, stderr.String())
		}
	}
}

func TestRunPinnedRoleReplacesArgv0(t *testing.T) {
	fake := newFakeTable("service", "alpha")
	var stderr bytes.Buffer
	if status := run("alpha", []string{"bin/example-alpha"}, fake.table, &stderr); status != 7 {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
	if !slices.Equal(fake.calls, []string{"alpha"}) {
		t.Fatalf("calls=%v", fake.calls)
	}

	fake = newFakeTable("service", "alpha")
	if status := run("service", nil, fake.table, &stderr); status != 7 || len(fake.args[0]) != 0 {
		t.Fatalf("pinned with empty argv: status=%d args=%v", status, fake.args)
	}
}

func TestRunRejectsNamesWithoutExactMatch(t *testing.T) {
	tests := []struct {
		name   string
		pinned string
		argv   []string
		want   string
	}{
		{name: "empty argv", argv: nil, want: ""},
		{name: "empty argv0", argv: []string{""}, want: ""},
		{name: "trailing slash", argv: []string{"/alpha/"}, want: ""},
		{name: "unselected role", argv: []string{"/beta"}, want: "beta"},
		{name: "case differs", argv: []string{"/Service"}, want: "Service"},
		{name: "extension", argv: []string{"/service.exe"}, want: "service.exe"},
		{name: "prefix", argv: []string{"/xservice"}, want: "xservice"},
		{name: "suffix", argv: []string{"/service-"}, want: "service-"},
		{name: "local artifact name", argv: []string{"bin/example-alpha"}, want: "example-alpha"},
		{name: "space", argv: []string{"/service "}, want: "service "},
		{name: "pinned unselected role", pinned: "beta", argv: []string{"/service"}, want: "beta"},
		{name: "newline", argv: []string{"/a\nb"}, want: "a\nb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeTable("service", "alpha")
			var stderr bytes.Buffer
			if status := run(tt.pinned, tt.argv, fake.table, &stderr); status != 2 {
				t.Fatalf("status=%d, want 2", status)
			}
			if len(fake.calls) != 0 {
				t.Fatalf("role code ran: %v", fake.calls)
			}
			want := `entrypoint: unknown role "` + strings.ReplaceAll(tt.want, "\n", `\n`) + `"; selected roles: service, alpha` + "\n"
			if stderr.String() != want {
				t.Fatalf("stderr=%q, want %q", stderr.String(), want)
			}
		})
	}
}

func TestRoleTable(t *testing.T) {
	if len(roles) == 0 || roles[0].name != "service" {
		t.Fatalf("first role must be service: %v", roles)
	}
	seen := map[string]bool{}
	for _, candidate := range roles {
		if candidate.name == "" || strings.ContainsAny(candidate.name, "/ \n") || candidate.main == nil {
			t.Fatalf("invalid role entry %+v", candidate)
		}
		if seen[candidate.name] {
			t.Fatalf("duplicate role %q", candidate.name)
		}
		seen[candidate.name] = true
	}
}

// TestUnknownNameProcessRejectsBeforeRoleCode runs the real entrypoint with an
// unmatched argv[0], a missing config file, and a role environment variable.
// A role would act on either; the rejection must produce only its stderr line.
func TestUnknownNameProcessRejectsBeforeRoleCode(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), executable)
	cmd.Args = []string{"/no-such-role", "-config", "/nonexistent/config.yaml"}
	cmd.Env = append(os.Environ(), execMainEnv+"=1", "APP__LOG__LEVEL=not-a-level")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("exit=%v, want status 2", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout=%q, want empty", stdout.String())
	}
	names := make([]string, 0, len(roles))
	for _, candidate := range roles {
		names = append(names, candidate.name)
	}
	want := `entrypoint: unknown role "no-such-role"; selected roles: ` + strings.Join(names, ", ") + "\n"
	if stderr.String() != want {
		t.Fatalf("stderr=%q, want %q", stderr.String(), want)
	}
}

// TestServiceNameReachesServiceRole proves argv[0] dispatch reaches a real
// role: the service role rejects an unknown flag before any other action.
func TestServiceNameReachesServiceRole(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), executable)
	cmd.Args = []string{"/service", "-no-such-flag"}
	cmd.Env = append(os.Environ(), execMainEnv+"=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("exit=%v stderr=%q, want status 1", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "unknown role") || !strings.Contains(stderr.String(), "no-such-flag") {
		t.Fatalf("stderr=%q, want the service role's flag rejection", stderr.String())
	}
}
