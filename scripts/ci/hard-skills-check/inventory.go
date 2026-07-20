package main

var domainSkills = []string{
	"go-api-contract",
	"go-chi",
	"go-concurrency",
	"go-data-architecture",
	"go-db-cache",
	"go-delivery-platform",
	"go-distributed",
	"go-domain-invariant",
	"go-idiomatic",
	"go-implementation-ownership",
	"go-language-simplifier",
	"go-observability",
	"go-performance",
	"go-reliability",
	"go-security",
	"go-structural-quality",
	"go-system-architecture",
	"go-test-strategy",
}

var targetSkills = []string{
	"go-api-contract",
	"go-chi",
	"go-coder",
	"go-concurrency",
	"go-data-architecture",
	"go-db-cache",
	"go-delivery-platform",
	"go-distributed",
	"go-domain-invariant",
	"go-idiomatic",
	"go-implementation-ownership",
	"go-language-simplifier",
	"go-observability",
	"go-performance",
	"go-reliability",
	"go-security",
	"go-structural-quality",
	"go-system-architecture",
	"go-systematic-debugging",
	"go-test-implementation",
	"go-test-strategy",
	"go-verification-before-completion",
}

var retiredSkills = []string{
	"api-contract-designer-spec",
	"go-api-contract-spec",
	"go-architect-spec",
	"go-chi-review",
	"go-chi-spec",
	"go-concurrency-review",
	"go-data-architect-spec",
	"go-data-architecture-spec",
	"go-db-cache-review",
	"go-db-cache-spec",
	"go-delivery-platform-review",
	"go-delivery-platform-spec",
	"go-design-review",
	"go-design-spec",
	"go-devops-review",
	"go-devops-spec",
	"go-distributed-architect-spec",
	"go-distributed-review",
	"go-distributed-spec",
	"go-domain-invariant-review",
	"go-domain-invariant-spec",
	"go-idiomatic-review",
	"go-implementation-ownership-review",
	"go-implementation-ownership-spec",
	"go-language-simplifier-review",
	"go-observability-engineer-spec",
	"go-observability-review",
	"go-observability-spec",
	"go-performance-review",
	"go-performance-spec",
	"go-qa-review",
	"go-qa-tester",
	"go-qa-tester-spec",
	"go-reliability-review",
	"go-reliability-spec",
	"go-security-review",
	"go-security-spec",
	"go-specialist-router",
	"go-structural-quality-review",
	"go-system-architecture-spec",
	"go-test-design",
	"go-test-review",
}

var selectedSkills = []string{
	"go-reliability",
	"go-security",
	"go-observability",
	"go-coder",
	"go-verification-before-completion",
	"go-implementation-ownership",
	"go-systematic-debugging",
	"go-concurrency",
	"go-test-strategy",
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
	return stringSet(targetSkills)
}

func retiredSkillSet() map[string]bool {
	return stringSet(retiredSkills)
}

func stringSet(names []string) map[string]bool {
	result := make(map[string]bool, len(names))
	for _, name := range names {
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
