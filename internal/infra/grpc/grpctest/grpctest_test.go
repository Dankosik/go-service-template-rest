package grpctest

import "testing"

func TestSplitFullMethodRejectsNonCanonicalPaths(t *testing.T) {
	service, method := splitFullMethod("/package.Service/Method")
	if service != "package.Service" || method != "Method" {
		t.Fatalf("splitFullMethod() = (%q, %q)", service, method)
	}

	for _, testCase := range []struct {
		name       string
		fullMethod string
	}{
		{name: "missing leading slash", fullMethod: "package.Service/Method"},
		{name: "missing service", fullMethod: "//Method"},
		{name: "missing method", fullMethod: "/package.Service/"},
		{name: "extra path component", fullMethod: "/package.Service/Method/Extra"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Fatalf("splitFullMethod(%q) did not panic", testCase.fullMethod)
				}
			}()
			splitFullMethod(testCase.fullMethod)
		})
	}
}
