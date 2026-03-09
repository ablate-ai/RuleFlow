package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func getRuleTemplateFilePath() string {
	return ResolveProjectPath("rules/template.yaml")
}

func ResolveProjectPath(path string) string {
	candidates := []string{
		path,
		filepath.Join("..", "..", path),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return path
}

// --- yaml.Node 操作辅助函数 ---

// yamlMappingNode 从文档节点取出顶层 MappingNode
func yamlMappingNode(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	if doc.Kind == yaml.MappingNode {
		return doc
	}
	return nil
}

// yamlFindInMapping 在 MappingNode 中查找 key，返回 value 节点（未找到返回 nil）
func yamlFindInMapping(mapping *yaml.Node, key string) *yaml.Node {
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

// yamlHasKey 检查 MappingNode 中是否存在某个 key
func yamlHasKey(mapping *yaml.Node, key string) bool {
	return yamlFindInMapping(mapping, key) != nil
}

// yamlDeleteFromMapping 从 MappingNode 中删除指定 key（及其 value）
func yamlDeleteFromMapping(mapping *yaml.Node, key string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}

// yamlSetInMapping 在 MappingNode 中设置或新增 key-value
func yamlSetInMapping(mapping *yaml.Node, key string, value *yaml.Node) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = value
			return
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
	mapping.Content = append(mapping.Content, keyNode, value)
}

// yamlValueToNode 将任意 Go 值序列化为 yaml.Node（用于将修改后的数据写回文档树）
func yamlValueToNode(v interface{}) (*yaml.Node, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0], nil
	}
	return &doc, nil
}

// adaptConfigForTargetNode 根据目标客户端类型调整 yaml.Node 树中的配置
// 使用 yaml.Node 操作，保留原始 YAML 格式（包括引号风格）
func adaptConfigForTargetNode(mapping *yaml.Node, target string) {
	if target != "stash" {
		return
	}
	// 移除 Clash 专属字段
	for _, key := range []string{"port", "socks-port", "redir-port", "mixed-port", "allow-lan", "external-controller", "secret", "tun"} {
		yamlDeleteFromMapping(mapping, key)
	}
	// 调整 DNS 配置
	dnsNode := yamlFindInMapping(mapping, "dns")
	if dnsNode != nil && dnsNode.Kind == yaml.MappingNode {
		yamlDeleteFromMapping(dnsNode, "prefer-h3")
		yamlDeleteFromMapping(dnsNode, "fake-ip-filter-mode")
		if !yamlHasKey(dnsNode, "fake-ip-filter") {
			filterNode, _ := yamlValueToNode([]string{"*.lan", "*.local"})
			yamlSetInMapping(dnsNode, "fake-ip-filter", filterNode)
		}
	}
}

// adaptConfigForTarget 根据目标客户端类型调整配置（仅用于 BuildYAMLFromDefaultTemplate）
func adaptConfigForTarget(cfg map[string]interface{}, target string) {
	if target == "stash" {
		// 移除 Clash 特定的端口设置
		delete(cfg, "port")
		delete(cfg, "socks-port")
		delete(cfg, "redir-port")
		delete(cfg, "mixed-port")
		delete(cfg, "allow-lan")
		delete(cfg, "external-controller")
		delete(cfg, "secret")
		delete(cfg, "tun")

		if dns, ok := cfg["dns"].(map[string]interface{}); ok {
			delete(dns, "prefer-h3")
			delete(dns, "fake-ip-filter-mode")
			if _, hasFakeIP := dns["fake-ip-filter"]; !hasFakeIP {
				dns["fake-ip-filter"] = []string{"*.lan", "*.local"}
			}
		}
	}
}

func adaptTemplateProxyGroups(raw interface{}, nodeNames []string) ([]interface{}, error) {
	groupList, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("proxy-groups 格式无效")
	}

	known := make(map[string]struct{}, len(nodeNames)+len(groupList))
	for _, n := range nodeNames {
		known[n] = struct{}{}
	}

	for _, item := range groupList {
		groupMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := groupMap["name"].(string)
		if name != "" {
			known[name] = struct{}{}
		}
	}

	for i, item := range groupList {
		groupMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		groupType, _ := groupMap["type"].(string)
		rawProxies, ok := groupMap["proxies"].([]interface{})
		if !ok {
			continue
		}

		filtered := make([]string, 0, len(rawProxies))
		for _, p := range rawProxies {
			name, ok := p.(string)
			if !ok || strings.TrimSpace(name) == "" {
				continue
			}

			if name == "__NODES__" {
				filtered = append(filtered, nodeNames...)
				continue
			}
			if _, exists := known[name]; exists || builtInProxyName(name) {
				filtered = append(filtered, name)
			}
		}

		if len(filtered) == 0 {
			switch strings.ToLower(groupType) {
			case "select", "url-test", "fallback", "load-balance":
				if len(nodeNames) > 0 {
					filtered = append(filtered, nodeNames...)
				} else {
					filtered = append(filtered, "DIRECT")
				}
			default:
				filtered = append(filtered, "DIRECT")
			}
		}

		filtered = dedupeStrings(filtered)
		converted := make([]interface{}, 0, len(filtered))
		for _, p := range filtered {
			converted = append(converted, p)
		}
		groupMap["proxies"] = converted
		groupList[i] = groupMap
	}

	return groupList, nil
}

func buildYAMLFromSourceTemplate(nodes []*ProxyNode, sourcePath string, target string) (string, error) {
	// 验证目标类型
	if target != "clash" && target != "stash" {
		return "", fmt.Errorf("不支持的目标类型: %s (支持: clash, stash)", target)
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败: %w", err)
	}

	// 用 yaml.Node 解析，保留原始格式（含引号风格）
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("解析模板文件失败: %w", err)
	}
	mapping := yamlMappingNode(&doc)
	if mapping == nil {
		return "", fmt.Errorf("模板文件格式无效：根节点不是映射")
	}

	// 根据目标类型调整配置（直接操作 yaml.Node，不破坏其他节点的格式）
	adaptConfigForTargetNode(mapping, target)

	proxies, nodeNames := buildProxies(nodes)
	proxiesNode, err := yamlValueToNode(proxies)
	if err != nil {
		return "", fmt.Errorf("生成 proxies 失败: %w", err)
	}
	yamlSetInMapping(mapping, "proxies", proxiesNode)

	rawGroupsNode := yamlFindInMapping(mapping, "proxy-groups")
	if rawGroupsNode != nil {
		var rawGroups interface{}
		if err := rawGroupsNode.Decode(&rawGroups); err == nil {
			adaptedGroups, err := adaptTemplateProxyGroups(rawGroups, nodeNames)
			if err != nil {
				return "", err
			}
			groupsNode, err := yamlValueToNode(adaptedGroups)
			if err != nil {
				return "", fmt.Errorf("生成 proxy-groups 失败: %w", err)
			}
			yamlSetInMapping(mapping, "proxy-groups", groupsNode)
		}
	} else {
		defaultGroups := []Group{
			{Name: "🚀 节点选择", Type: "select", Proxies: append([]string{"♻️ 自动选择", "DIRECT"}, nodeNames...)},
			{Name: "♻️ 自动选择", Type: "url-test", Proxies: nodeNames},
		}
		groupsNode, err := yamlValueToNode(defaultGroups)
		if err != nil {
			return "", fmt.Errorf("生成 proxy-groups 失败: %w", err)
		}
		yamlSetInMapping(mapping, "proxy-groups", groupsNode)
	}

	if !yamlHasKey(mapping, "rules") {
		rulesNode, err := yamlValueToNode(cloneRules(defaultRules))
		if err != nil {
			return "", fmt.Errorf("生成 rules 失败: %w", err)
		}
		yamlSetInMapping(mapping, "rules", rulesNode)
	}

	yamlData, err := yaml.Marshal(&doc)
	if err != nil {
		return "", fmt.Errorf("生成配置失败")
	}
	return string(yamlData), nil
}

// BuildYAMLFromTemplateContent 从模板内容（字符串）构建 YAML 配置
func BuildYAMLFromTemplateContent(nodes []*ProxyNode, templateContent string, target string) (string, error) {
	if target != "clash" && target != "stash" {
		return "", fmt.Errorf("不支持的目标类型: %s (支持: clash, stash)", target)
	}

	// 用 yaml.Node 解析，保留原始格式（含引号风格）
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(templateContent), &doc); err != nil {
		return "", fmt.Errorf("解析模板内容失败: %w", err)
	}
	mapping := yamlMappingNode(&doc)
	if mapping == nil {
		return "", fmt.Errorf("模板内容格式无效：根节点不是映射")
	}

	adaptConfigForTargetNode(mapping, target)

	proxies, nodeNames := buildProxies(nodes)
	proxiesNode, err := yamlValueToNode(proxies)
	if err != nil {
		return "", fmt.Errorf("生成 proxies 失败: %w", err)
	}
	yamlSetInMapping(mapping, "proxies", proxiesNode)

	rawGroupsNode := yamlFindInMapping(mapping, "proxy-groups")
	if rawGroupsNode != nil {
		var rawGroups interface{}
		if err := rawGroupsNode.Decode(&rawGroups); err == nil {
			adaptedGroups, err := adaptTemplateProxyGroups(rawGroups, nodeNames)
			if err != nil {
				return "", err
			}
			groupsNode, err := yamlValueToNode(adaptedGroups)
			if err != nil {
				return "", fmt.Errorf("生成 proxy-groups 失败: %w", err)
			}
			yamlSetInMapping(mapping, "proxy-groups", groupsNode)
		}
	} else {
		defaultGroups := []Group{
			{Name: "🚀 节点选择", Type: "select", Proxies: append([]string{"♻️ 自动选择", "DIRECT"}, nodeNames...)},
			{Name: "♻️ 自动选择", Type: "url-test", Proxies: nodeNames},
		}
		groupsNode, err := yamlValueToNode(defaultGroups)
		if err != nil {
			return "", fmt.Errorf("生成 proxy-groups 失败: %w", err)
		}
		yamlSetInMapping(mapping, "proxy-groups", groupsNode)
	}

	if !yamlHasKey(mapping, "rules") {
		rulesNode, err := yamlValueToNode(cloneRules(defaultRules))
		if err != nil {
			return "", fmt.Errorf("生成 rules 失败: %w", err)
		}
		yamlSetInMapping(mapping, "rules", rulesNode)
	}

	yamlData, err := yaml.Marshal(&doc)
	if err != nil {
		return "", fmt.Errorf("生成配置失败")
	}
	return string(yamlData), nil
}

// BuildYAMLFromDefaultTemplate 使用内置默认规则生成 YAML（无模板时使用）
func BuildYAMLFromDefaultTemplate(nodes []*ProxyNode, target string) (string, error) {
	if target != "clash" && target != "stash" {
		return "", fmt.Errorf("不支持的目标类型: %s (支持: clash, stash)", target)
	}

	cfg := map[string]interface{}{}
	adaptConfigForTarget(cfg, target)

	proxies, nodeNames := buildProxies(nodes)
	cfg["proxies"] = proxies
	cfg["proxy-groups"] = []Group{
		{
			Name:    "🚀 节点选择",
			Type:    "select",
			Proxies: append([]string{"♻️ 自动选择", "DIRECT"}, nodeNames...),
		},
		{
			Name:    "♻️ 自动选择",
			Type:    "url-test",
			Proxies: nodeNames,
		},
	}
	cfg["rules"] = cloneRules(defaultRules)

	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("生成配置失败")
	}
	return string(yamlData), nil
}

// buildYAMLFromSourceTemplateWithTrojan 从 Trojan 节点构建配置（向后兼容）
func buildYAMLFromSourceTemplateWithTrojan(nodes []TrojanNode, sourcePath string, target string) (string, error) {
	// 转换为 ProxyNode
	proxyNodes := make([]*ProxyNode, len(nodes))
	for i, node := range nodes {
		proxyNodes[i] = &ProxyNode{
			Protocol: "trojan",
			Name:     node.Name,
			Server:   node.Server,
			Port:     node.Port,
			Options: map[string]interface{}{
				"password": node.Password,
				"sni":      node.SNI,
			},
		}
	}
	return buildYAMLFromSourceTemplate(proxyNodes, sourcePath, target)
}
