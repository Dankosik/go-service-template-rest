package natsjstest

import "testing"

func TestRequireDocker(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty"},
		{name: "zero", value: "0"},
		{name: "false", value: "false"},
		{name: "one", value: "1", want: true},
		{name: "trimmed true", value: " true ", want: true},
		{name: "case insensitive yes", value: "YES", want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("REQUIRE_DOCKER", test.value)
			if got := requireDocker(); got != test.want {
				t.Fatalf("requireDocker() = %t, want %t", got, test.want)
			}
		})
	}
}
