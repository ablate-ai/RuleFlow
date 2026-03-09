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

// adaptConfigForTarget 根据目标客户端类型调整配置
func adaptConfigForTarget(cfg map[string]interface{}, target string) {
	if target == "stash" {
		// Stash 特定的配置调整
		// 移除 Clash 特定的端口设置
		delete(cfg, "port")
		delete(cfg, "socks-port")
		delete(cfg, "redir-port")
		delete(cfg, "mixed-port")
		delete(cfg, "allow-lan")
		delete(cfg, "external-controller")
		delete(cfg, "secret")

		// 移除 TUN 配置（Stash 在 iOS 上不支持）
		delete(cfg, "tun")

		// 调整 DNS 配置以兼容 Stash
		if dns, ok := cfg["dns"].(map[string]interface{}); ok {
			// 移除 Clash 特定的 DNS 设置
			delete(dns, "prefer-h3")
			delete(dns, "fake-ip-filter-mode")

			// 确保 fake-ip-filter 存在且格式正确
			if _, hasFakeIP := dns["fake-ip-filter"]; !hasFakeIP {
				dns["fake-ip-filter"] = []string{"*.lan", "*.local"}
			}
		}

		// 设置 allow-lan 为 false（已删除，不再设置）
		// Stash 不需要这个配置
	}
	// 对于 Clash，保持原有配置不变
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

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("解析模板文件失败: %w", err)
	}

	// 根据目标类型调整配置
	adaptConfigForTarget(cfg, target)

	proxies, nodeNames := buildProxies(nodes)
	cfg["proxies"] = proxies

	rawGroups, ok := cfg["proxy-groups"]
	if ok {
		adaptedGroups, err := adaptTemplateProxyGroups(rawGroups, nodeNames)
		if err != nil {
			return "", err
		}
		cfg["proxy-groups"] = adaptedGroups
	} else {
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
	}

	if _, ok := cfg["rules"]; !ok {
		cfg["rules"] = cloneRules(defaultRules)
	}

	yamlData, err := yaml.Marshal(cfg)
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

	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(templateContent), &cfg); err != nil {
		return "", fmt.Errorf("解析模板内容失败: %w", err)
	}

	adaptConfigForTarget(cfg, target)

	proxies, nodeNames := buildProxies(nodes)
	cfg["proxies"] = proxies

	rawGroups, ok := cfg["proxy-groups"]
	if ok {
		adaptedGroups, err := adaptTemplateProxyGroups(rawGroups, nodeNames)
		if err != nil {
			return "", err
		}
		cfg["proxy-groups"] = adaptedGroups
	} else {
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
	}

	if _, ok := cfg["rules"]; !ok {
		cfg["rules"] = cloneRules(defaultRules)
	}

	yamlData, err := yaml.Marshal(cfg)
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
