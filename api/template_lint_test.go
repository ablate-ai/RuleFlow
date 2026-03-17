package api

import (
	"context"
	"strings"
	"testing"

	"github.com/ablate-ai/RuleFlow/internal/app"
)

func TestLintRuleEntriesDetectsDuplicatesAndTerminalOrder(t *testing.T) {
	entries := buildRuleEntries([]string{
		"DOMAIN-SUFFIX,example.com,Proxy",
		"MATCH,DIRECT",
		"DOMAIN-SUFFIX,example.com,Proxy",
		"RULE-SET,https://example.com/list,Proxy",
	})

	warnings := lintRuleEntries(entries)
	joined := strings.Join(warnings, "\n")

	if !strings.Contains(joined, "重复") {
		t.Fatalf("期望检测到重复规则，实际 warnings=%v", warnings)
	}
	if !strings.Contains(joined, "不会生效") {
		t.Fatalf("期望检测到兜底规则后仍有规则，实际 warnings=%v", warnings)
	}
}

func TestLintRuleEntriesDetectsMissingTerminalRule(t *testing.T) {
	entries := buildRuleEntries([]string{
		"DOMAIN-SUFFIX,example.com,Proxy",
		"RULE-SET,https://example.com/list,Proxy",
	})

	warnings := lintRuleEntries(entries)
	joined := strings.Join(warnings, "\n")

	if !strings.Contains(joined, "缺少兜底规则") {
		t.Fatalf("期望检测到缺少兜底规则，实际 warnings=%v", warnings)
	}
}

func TestExtractSurgeRules(t *testing.T) {
	content := `
[General]
loglevel = notify

[Rule]
DOMAIN,example.com,DIRECT
FINAL,Proxy

[Host]
`

	rules := extractSurgeRules(content)
	if len(rules) != 2 {
		t.Fatalf("期望提取 2 条规则，实际为 %d: %#v", len(rules), rules)
	}
	if rules[0] != "DOMAIN,example.com,DIRECT" || rules[1] != "FINAL,Proxy" {
		t.Fatalf("提取的规则不符合预期: %#v", rules)
	}
}

func TestExtractYAMLRules(t *testing.T) {
	content := `
proxies: []
rules:
  - DOMAIN,example.com,DIRECT
  - MATCH,Proxy
`

	rules, err := extractYAMLRules(content)
	if err != nil {
		t.Fatalf("extractYAMLRules() 返回错误: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("期望提取 2 条规则，实际为 %d: %#v", len(rules), rules)
	}
}

func TestExpandRuleSetRules(t *testing.T) {
	ref := ruleSetReference{
		ParentIndex: 3,
		Policy:      "Proxy",
		Source:      "/rulesets/demo?target=surge",
	}
	rules := []app.RuleSetRule{
		{Type: "domain_suffix", Value: "example.com"},
		{Type: "ip_cidr", Value: "1.1.1.0/24", NoResolve: true},
	}

	got := expandRuleSetRules(ref, rules)
	if len(got) != 2 {
		t.Fatalf("期望展开 2 条规则，实际为 %d: %#v", len(got), got)
	}
	if got[0].ParentIndex != 3 || got[0].Policy != "Proxy" || got[0].Kind != "DOMAIN-SUFFIX" {
		t.Fatalf("第一条展开结果不符合预期: %#v", got[0])
	}
	if got[1].Kind != "IP-CIDR" || !got[1].NoResolve {
		t.Fatalf("第二条展开结果不符合预期: %#v", got[1])
	}
}

func TestLintExpandedRulesDetectsConflict(t *testing.T) {
	warnings := lintExpandedRules([]expandedRule{
		{ParentIndex: 1, Kind: "DOMAIN-SUFFIX", Matcher: "example.com", Policy: "DIRECT"},
		{ParentIndex: 2, Kind: "DOMAIN-SUFFIX", Matcher: "example.com", Policy: "Proxy"},
	})

	if joined := strings.Join(warnings, "\n"); !strings.Contains(joined, "规则冲突") {
		t.Fatalf("期望检测到展开后的规则冲突，实际 warnings=%v", warnings)
	}
}

func TestExpandRuleSetReferencesNoRefs(t *testing.T) {
	got, warnings := expandRuleSetReferences(context.Background(), &Handlers{}, nil)
	if len(got) != 0 || len(warnings) != 0 {
		t.Fatalf("空引用不应产生结果，got=%#v warnings=%v", got, warnings)
	}
}
