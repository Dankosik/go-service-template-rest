package main

var targetSkills = []string{
	"go-api-contract-spec",
	"go-chi-review",
	"go-chi-spec",
	"go-coder",
	"go-concurrency-review",
	"go-data-architecture-spec",
	"go-db-cache-review",
	"go-db-cache-spec",
	"go-delivery-platform-review",
	"go-delivery-platform-spec",
	"go-distributed-review",
	"go-distributed-spec",
	"go-domain-invariant-review",
	"go-domain-invariant-spec",
	"go-idiomatic-review",
	"go-implementation-ownership-review",
	"go-implementation-ownership-spec",
	"go-language-simplifier-review",
	"go-observability-review",
	"go-observability-spec",
	"go-performance-review",
	"go-performance-spec",
	"go-reliability-review",
	"go-reliability-spec",
	"go-security-review",
	"go-security-spec",
	"go-structural-quality-review",
	"go-system-architecture-spec",
	"go-systematic-debugging",
	"go-test-design",
	"go-test-implementation",
	"go-test-review",
	"go-verification-before-completion",
	"go-specialist-router",
}

var renamedSkills = map[string]string{
	"go-api-contract-spec":               "api-contract-designer-spec",
	"go-system-architecture-spec":        "go-architect-spec",
	"go-data-architecture-spec":          "go-data-architect-spec",
	"go-implementation-ownership-spec":   "go-design-spec",
	"go-implementation-ownership-review": "go-design-review",
	"go-distributed-spec":                "go-distributed-architect-spec",
	"go-observability-spec":              "go-observability-engineer-spec",
	"go-delivery-platform-spec":          "go-devops-spec",
	"go-delivery-platform-review":        "go-devops-review",
	"go-test-design":                     "go-qa-tester-spec",
	"go-test-implementation":             "go-qa-tester",
	"go-test-review":                     "go-qa-review",
}

var selectedSkills = []string{
	"go-reliability-review",
	"go-security-spec",
	"go-observability-review",
	"go-coder",
	"go-verification-before-completion",
	"go-security-review",
	"go-implementation-ownership-review",
	"go-systematic-debugging",
	"go-specialist-router",
}

var evalCategories = []string{
	"domain_defect",
	"clean",
	"neighbor",
	"unresolved_policy",
}

var executionSkills = map[string]bool{
	"go-coder":                          true,
	"go-systematic-debugging":           true,
	"go-test-implementation":            true,
	"go-verification-before-completion": true,
}

func targetSkillSet() map[string]bool {
	result := make(map[string]bool, len(targetSkills))
	for _, name := range targetSkills {
		result[name] = true
	}
	return result
}

func retiredSkillSet() map[string]bool {
	result := make(map[string]bool, len(renamedSkills))
	for _, name := range renamedSkills {
		result[name] = true
	}
	return result
}

func isSelectedSkill(name string) bool {
	for _, selected := range selectedSkills {
		if name == selected {
			return true
		}
	}
	return false
}
