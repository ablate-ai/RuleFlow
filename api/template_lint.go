package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ablate-ai/RuleFlow/internal/app"
	"gopkg.in/yaml.v3"
)

type templateValidationResult struct {
	Message  string   `json:"message"`
	Warnings []string `json:"warnings,omitempty"`
}

type templateRuleEntry struct {
	Index int
	Kind  string
	Raw   string
	Norm  string
}

type expandedRule struct {
	ParentIndex int
	Kind        string
	Matcher     string
	Policy      string
	NoResolve   bool
	Source      string
}

type ruleSetReference struct {
	ParentIndex int
	Kind        string
	Policy      string
	Source      string
	Format      string
	Raw         string
}

type yamlRuleProvider struct {
	URL    string
	Format string
}

func lintGeneratedTemplate(ctx context.Context, h *Handlers, target, configContent string) []string {
	switch target {
	case "clash-meta", "stash":
		return lintYAMLTemplate(ctx, h, configContent)
	case "surge":
		return lintSurgeTemplate(ctx, h, configContent)
	default:
		return nil
	}
}

func lintYAMLTemplate(ctx context.Context, h *Handlers, configContent string) []string {
	rules, err := extractYAMLRules(configContent)
	if err != nil {
		return []string{fmt.Sprintf("无法分析 rules 顺序: %v", err)}
	}

	warnings := lintRuleEntries(buildRuleEntries(rules))
	providers, err := extractYAMLRuleProviders(configContent)
	if err != nil {
		return append(warnings, fmt.Sprintf("无法分析 rule-providers: %v", err))
	}

	return append(warnings, lintExpandedRuleSets(ctx, h, buildYAMLRuleSetRefs(rules, providers))...)
}

func lintSurgeTemplate(ctx context.Context, h *Handlers, configContent string) []string {
	rules := extractSurgeRules(configContent)
	warnings := lintRuleEntries(buildRuleEntries(rules))
	return append(warnings, lintExpandedRuleSets(ctx, h, buildSurgeRuleSetRefs(rules))...)
}

func lintExpandedRuleSets(ctx context.Context, h *Handlers, refs []ruleSetReference) []string {
	if len(refs) == 0 {
		return nil
	}

	if h == nil {
		return []string{"规则展开检查不可用：处理器未初始化"}
	}

	expanded, warnings := expandRuleSetReferences(ctx, h, refs)
	return append(warnings, lintExpandedRules(expanded)...)
}

func expandRuleSetReferences(ctx context.Context, h *Handlers, refs []ruleSetReference) ([]expandedRule, []string) {
	expanded := make([]expandedRule, 0)
	warnings := make([]string, 0)

	for _, ref := range refs {
		rules, err := loadRuleSetRules(ctx, h, ref)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("第 %d 条规则引用的 ruleset 无法展开：%s（%v）", ref.ParentIndex, ref.Source, err))
			continue
		}

		items := expandRuleSetRules(ref, rules)
		expanded = append(expanded, items...)
	}

	return expanded, warnings
}

func loadRuleSetRules(ctx context.Context, h *Handlers, ref ruleSetReference) ([]app.RuleSetRule, error) {
	if strings.HasPrefix(ref.Source, "/rulesets/") {
		return loadLocalRuleSetRules(ctx, h, ref.Source)
	}

	if strings.HasPrefix(ref.Source, "http://") || strings.HasPrefix(ref.Source, "https://") {
		return loadRemoteRuleSetRules(ctx, ref)
	}

	return nil, fmt.Errorf("暂不支持的 ruleset 来源")
}

func loadLocalRuleSetRules(ctx context.Context, h *Handlers, source string) ([]app.RuleSetRule, error) {
	if h.ruleSourceService == nil {
		return nil, fmt.Errorf("规则源服务不可用")
	}

	u, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("ruleset 路径无效: %w", err)
	}

	name := strings.TrimPrefix(u.Path, "/rulesets/")
	name, err = url.PathUnescape(name)
	if err != nil {
		return nil, fmt.Errorf("ruleset 名称解码失败: %w", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("ruleset 名称为空")
	}

	sourceObj, err := h.ruleSourceService.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(sourceObj.ParsedRules) == 0 {
		return nil, fmt.Errorf("规则源尚未同步")
	}

	var rules []app.RuleSetRule
	if err := json.Unmarshal(sourceObj.ParsedRules, &rules); err != nil {
		return nil, fmt.Errorf("读取已同步规则失败: %w", err)
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("规则源没有可用规则")
	}

	return rules, nil
}

func loadRemoteRuleSetRules(ctx context.Context, ref ruleSetReference) ([]app.RuleSetRule, error) {
	format := strings.TrimSpace(ref.Format)
	if format == "" {
		return nil, fmt.Errorf("无法判断规则集格式")
	}

	content, _, err := app.FetchSubscriptionContent(ctx, ref.Source)
	if err != nil {
		return nil, err
	}

	rules, err := app.ParseRuleSet(content, format)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("规则集为空")
	}

	return rules, nil
}

func expandedRuleFromRuleSet(ref ruleSetReference, rule app.RuleSetRule) (expandedRule, bool) {
	kind := ""
	switch rule.Type {
	case "domain_suffix":
		kind = "DOMAIN-SUFFIX"
	case "domain_keyword":
		kind = "DOMAIN-KEYWORD"
	case "domain":
		kind = "DOMAIN"
	case "ip_cidr":
		kind = "IP-CIDR"
	default:
		return expandedRule{}, false
	}

	return expandedRule{
		ParentIndex: ref.ParentIndex,
		Kind:        kind,
		Matcher:     strings.TrimSpace(rule.Value),
		Policy:      strings.TrimSpace(ref.Policy),
		NoResolve:   rule.NoResolve,
		Source:      ref.Source,
	}, true
}

func expandRuleSetRules(ref ruleSetReference, rules []app.RuleSetRule) []expandedRule {
	expanded := make([]expandedRule, 0, len(rules))
	for _, rule := range rules {
		item, ok := expandedRuleFromRuleSet(ref, rule)
		if ok {
			expanded = append(expanded, item)
		}
	}
	return expanded
}

func lintExpandedRules(rules []expandedRule) []string {
	if len(rules) == 0 {
		return nil
	}

	type seenRule struct {
		ParentIndex int
		Policy      string
		Source      string
	}

	seen := make(map[string]seenRule, len(rules))
	warnings := make([]string, 0)

	for _, rule := range rules {
		matchKey := fmt.Sprintf("%s|%s|%t", rule.Kind, strings.ToLower(rule.Matcher), rule.NoResolve)
		effectiveKey := matchKey + "|" + strings.ToLower(rule.Policy)

		if prev, ok := seen[effectiveKey]; ok {
			warnings = append(warnings, fmt.Sprintf(
				"展开后发现重复规则：第 %d 条 ruleset 与第 %d 条 ruleset 都包含 %s,%s -> %s",
				rule.ParentIndex, prev.ParentIndex, rule.Kind, rule.Matcher, rule.Policy,
			))
			continue
		}

		if prev, ok := seen[matchKey]; ok && !strings.EqualFold(prev.Policy, rule.Policy) {
			warnings = append(warnings, fmt.Sprintf(
				"展开后发现规则冲突：第 %d 条 ruleset 中的 %s,%s -> %s 会覆盖第 %d 条 ruleset 中指向 %s 的同一匹配项",
				rule.ParentIndex, rule.Kind, rule.Matcher, rule.Policy, prev.ParentIndex, prev.Policy,
			))
		}

		seen[effectiveKey] = seenRule{
			ParentIndex: rule.ParentIndex,
			Policy:      rule.Policy,
			Source:      rule.Source,
		}
		if _, ok := seen[matchKey]; !ok {
			seen[matchKey] = seenRule{
				ParentIndex: rule.ParentIndex,
				Policy:      rule.Policy,
				Source:      rule.Source,
			}
		}
	}

	return dedupeStrings(warnings)
}

func extractYAMLRules(content string) ([]string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, err
	}

	mapping := yamlRootMappingNode(&doc)
	if mapping == nil {
		return nil, fmt.Errorf("根节点不是映射")
	}

	rulesNode := yamlLookupMappingValue(mapping, "rules")
	if rulesNode == nil {
		return nil, nil
	}
	if rulesNode.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("rules 不是序列")
	}

	rules := make([]string, 0, len(rulesNode.Content))
	for _, item := range rulesNode.Content {
		if item.Kind != yaml.ScalarNode {
			continue
		}
		rule := strings.TrimSpace(item.Value)
		if rule != "" {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func extractYAMLRuleProviders(content string) (map[string]yamlRuleProvider, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, err
	}

	mapping := yamlRootMappingNode(&doc)
	if mapping == nil {
		return nil, fmt.Errorf("根节点不是映射")
	}

	providersNode := yamlLookupMappingValue(mapping, "rule-providers")
	if providersNode == nil || providersNode.Kind != yaml.MappingNode {
		return map[string]yamlRuleProvider{}, nil
	}

	providers := make(map[string]yamlRuleProvider)
	for i := 0; i+1 < len(providersNode.Content); i += 2 {
		name := strings.TrimSpace(providersNode.Content[i].Value)
		valueNode := providersNode.Content[i+1]
		if name == "" || valueNode.Kind != yaml.MappingNode {
			continue
		}
		urlNode := yamlLookupMappingValue(valueNode, "url")
		behaviorNode := yamlLookupMappingValue(valueNode, "behavior")
		providers[name] = yamlRuleProvider{
			URL:    scalarNodeValue(urlNode),
			Format: providerBehaviorToFormat(scalarNodeValue(behaviorNode)),
		}
	}

	return providers, nil
}

func extractSurgeRules(content string) []string {
	lines := strings.Split(content, "\n")
	rules := make([]string, 0)
	inRuleSection := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inRuleSection = line == "[Rule]"
			continue
		}
		if !inRuleSection || line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		rules = append(rules, line)
	}

	return rules
}

func buildRuleEntries(rules []string) []templateRuleEntry {
	entries := make([]templateRuleEntry, 0, len(rules))
	for i, rule := range rules {
		normalized := normalizeRule(rule)
		if normalized == "" {
			continue
		}
		entries = append(entries, templateRuleEntry{
			Index: i + 1,
			Kind:  ruleKind(normalized),
			Raw:   rule,
			Norm:  normalized,
		})
	}
	return entries
}

func buildYAMLRuleSetRefs(rules []string, providers map[string]yamlRuleProvider) []ruleSetReference {
	refs := make([]ruleSetReference, 0)
	for i, rule := range rules {
		parts := splitCSVLine(rule)
		if len(parts) < 3 || !strings.EqualFold(strings.TrimSpace(parts[0]), "RULE-SET") {
			continue
		}

		providerName := strings.TrimSpace(parts[1])
		policy := strings.TrimSpace(parts[2])
		provider, ok := providers[providerName]
		if !ok || strings.TrimSpace(provider.URL) == "" {
			continue
		}

		refs = append(refs, ruleSetReference{
			ParentIndex: i + 1,
			Kind:        "RULE-SET",
			Policy:      policy,
			Source:      strings.TrimSpace(provider.URL),
			Format:      provider.Format,
			Raw:         rule,
		})
	}
	return refs
}

func buildSurgeRuleSetRefs(rules []string) []ruleSetReference {
	refs := make([]ruleSetReference, 0)
	for i, rule := range rules {
		parts := splitCSVLine(rule)
		if len(parts) < 3 {
			continue
		}

		kind := strings.ToUpper(strings.TrimSpace(parts[0]))
		if kind != "RULE-SET" && kind != "DOMAIN-SET" {
			continue
		}

		source := strings.TrimSpace(parts[1])
		policy := strings.TrimSpace(parts[2])
		refs = append(refs, ruleSetReference{
			ParentIndex: i + 1,
			Kind:        kind,
			Policy:      policy,
			Source:      source,
			Format:      inferRuleSetFormat(source, kind),
			Raw:         rule,
		})
	}
	return refs
}

func lintRuleEntries(entries []templateRuleEntry) []string {
	if len(entries) == 0 {
		return []string{"未检测到任何 rules，无法检查规则顺序与重复项"}
	}

	warnings := make([]string, 0)
	seen := make(map[string]templateRuleEntry, len(entries))
	terminalIndexes := make([]int, 0, 2)

	for _, entry := range entries {
		if prev, ok := seen[entry.Norm]; ok {
			warnings = append(warnings, fmt.Sprintf("第 %d 条规则与第 %d 条重复：%s", entry.Index, prev.Index, entry.Raw))
			continue
		}
		seen[entry.Norm] = entry

		if isTerminalRuleKind(entry.Kind) {
			terminalIndexes = append(terminalIndexes, entry.Index)
		}
	}

	if len(terminalIndexes) == 0 {
		warnings = append(warnings, "缺少兜底规则，建议最后加上 MATCH 或 FINAL")
		return warnings
	}

	if len(terminalIndexes) > 1 {
		warnings = append(warnings, fmt.Sprintf("检测到多个兜底规则：第 %d、%s 条", terminalIndexes[0], joinRuleIndexes(terminalIndexes[1:])))
	}

	lastTerminal := terminalIndexes[len(terminalIndexes)-1]
	if lastTerminal != len(entries) {
		warnings = append(warnings, fmt.Sprintf("兜底规则位于第 %d 条，后面还有 %d 条规则不会生效", lastTerminal, len(entries)-lastTerminal))
	}

	return warnings
}

func normalizeRule(rule string) string {
	parts := strings.Split(rule, ",")
	normalized := make([]string, 0, len(parts))
	for i, part := range parts {
		value := strings.TrimSpace(part)
		if i == 0 {
			value = strings.ToUpper(value)
		}
		normalized = append(normalized, value)
	}
	return strings.Join(normalized, ",")
}

func ruleKind(rule string) string {
	if idx := strings.IndexByte(rule, ','); idx >= 0 {
		return strings.ToUpper(strings.TrimSpace(rule[:idx]))
	}
	return strings.ToUpper(strings.TrimSpace(rule))
}

func isTerminalRuleKind(kind string) bool {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case "MATCH", "FINAL":
		return true
	default:
		return false
	}
}

func inferRuleSetFormat(source, kind string) string {
	if u, err := url.Parse(source); err == nil {
		target := strings.TrimSpace(u.Query().Get("target"))
		switch target {
		case "surge":
			return "surge"
		case "clash-classical":
			return "clash-classical"
		case "clash-domain":
			return "clash-domain"
		case "clash-ipcidr":
			return "clash-ipcidr"
		case "domain-list":
			return "domain-list"
		case "ip-list":
			return "ip-list"
		}
	}

	if strings.EqualFold(kind, "DOMAIN-SET") {
		return "domain-list"
	}
	return "surge"
}

func providerBehaviorToFormat(behavior string) string {
	switch strings.ToLower(strings.TrimSpace(behavior)) {
	case "domain":
		return "clash-domain"
	case "ipcidr":
		return "clash-ipcidr"
	case "classical":
		return "clash-classical"
	default:
		return ""
	}
}

func scalarNodeValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return strings.TrimSpace(node.Value)
}

func joinRuleIndexes(indexes []int) string {
	values := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		values = append(values, fmt.Sprintf("%d", idx))
	}
	return strings.Join(values, "、")
}

func yamlRootMappingNode(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return yamlRootMappingNode(doc.Content[0])
	}
	if doc.Kind == yaml.MappingNode {
		return doc
	}
	return nil
}

func yamlLookupMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func splitCSVLine(line string) []string {
	parts := strings.Split(line, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}
