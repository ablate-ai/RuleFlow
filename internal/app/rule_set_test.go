package app

import (
	"strings"
	"testing"
)

func TestParseRuleSetClassical(t *testing.T) {
	content := `
DOMAIN-SUFFIX,claude.ai,🤖 AI
DOMAIN,example.com,DIRECT
IP-CIDR,1.1.1.1/32,no-resolve
`

	rules, err := ParseRuleSet(content, "surge")
	if err != nil {
		t.Fatalf("ParseRuleSet() error = %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("ParseRuleSet() len = %d, want 3", len(rules))
	}
	if rules[0].Type != "domain_suffix" || rules[0].Value != "claude.ai" {
		t.Fatalf("unexpected first rule: %+v", rules[0])
	}
}

func TestExportRuleSetSingBox(t *testing.T) {
	rules := []RuleSetRule{
		{Type: "domain_suffix", Value: "claude.ai"},
		{Type: "domain", Value: "example.com"},
		{Type: "ip_cidr", Value: "1.1.1.1/32", NoResolve: true},
	}

	content, err := ExportRuleSet(rules, "sing-box")
	if err != nil {
		t.Fatalf("ExportRuleSet() error = %v", err)
	}
	if !strings.Contains(content, "\"domain_suffix\"") || !strings.Contains(content, "\"ip_cidr\"") {
		t.Fatalf("unexpected sing-box export: %s", content)
	}
}
