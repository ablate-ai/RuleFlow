package app

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBuildYAMLSupportsNegativeLookaheadFilter(t *testing.T) {
	templateContent := `proxy-groups:
  - name: Auto
    type: url-test
    filter: "^((?!v2).)*$"
    proxies: ["__NODES__"]
proxies: []
rules: []
`
	nodes := []*ProxyNode{
		{
			Protocol: "trojan",
			Name:     "SG Node 1",
			Server:   "sg1.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "password123",
				"sni":      "sg1.example.com",
			},
		},
		{
			Protocol: "trojan",
			Name:     "v2 SG Node",
			Server:   "sg2.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "password456",
				"sni":      "sg2.example.com",
			},
		},
	}

	config, err := BuildYAMLFromTemplateContent(nodes, templateContent, "clash-meta")
	if err != nil {
		t.Fatalf("YAML filter 负向前瞻生成失败: %v", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &cfg); err != nil {
		t.Fatalf("生成的 YAML 不是合法配置: %v", err)
	}
	groups, ok := cfg["proxy-groups"].([]interface{})
	if !ok || len(groups) == 0 {
		t.Fatalf("生成的 YAML 缺少 proxy-groups")
	}
	group, ok := groups[0].(map[string]interface{})
	if !ok {
		t.Fatalf("proxy-group 结构无效: %#v", groups[0])
	}
	proxies, ok := group["proxies"].([]interface{})
	if !ok {
		t.Fatalf("group proxies 结构无效: %#v", group["proxies"])
	}
	if len(proxies) != 1 || proxies[0] != "🇸🇬 SG Node 1" {
		t.Fatalf("期望负向前瞻排除 v2 节点，实际配置为:\n%s", config)
	}
}

func TestClashMetaProxyGroupsUseURL(t *testing.T) {
	templateContent := `proxy-groups:
  - type: select
    name: 📺 Stream
    benchmark-url: http://wifi.vivo.com.cn/generate_204
    proxies: ["🚀 Proxy", "🇺🇸 US", "🇯🇵 JP", "🇭🇰 HK", "🇸🇬 SG"]
  - type: url-test
    name: ⚡ Auto
    benchmark-url: http://wifi.vivo.com.cn/generate_204
    proxies: ["__NODES__"]
proxies: []
rules: []
`

	nodes := []*ProxyNode{
		{
			Protocol: "trojan",
			Name:     "us.hnl.qqpw",
			Server:   "us.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "password123",
				"sni":      "us.example.com",
			},
		},
	}

	config, err := BuildYAMLFromTemplateContent(nodes, templateContent, "clash-meta")
	if err != nil {
		t.Fatalf("生成 Clash Meta 配置失败: %v", err)
	}

	if strings.Contains(config, "benchmark-url: http://wifi.vivo.com.cn/generate_204") {
		t.Fatalf("Clash Meta proxy-groups 不应保留 benchmark-url 字段，实际配置为:\n%s", config)
	}
	if !strings.Contains(config, "url: http://wifi.vivo.com.cn/generate_204") {
		t.Fatalf("Clash Meta proxy-groups 应使用 url 字段，实际配置为:\n%s", config)
	}
}

func TestBuildYAMLPrunesRulesForMissingPolicy(t *testing.T) {
	templateContent := `proxy-groups:
  - type: select
    name: 🚀 Proxy
    proxies: ["DIRECT", "__NODES__"]
proxies: []
rules:
  - RULE-SET,wireguard-home,🏠 Home
  - RULE-SET,need_direct,DIRECT
`

	nodes := []*ProxyNode{
		{
			Protocol: "trojan",
			Name:     "US Node 1",
			Server:   "us1.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "password123",
				"sni":      "us1.example.com",
			},
		},
	}

	config, err := BuildYAMLFromTemplateContent(nodes, templateContent, "clash-meta")
	if err != nil {
		t.Fatalf("生成 Clash Meta 配置失败: %v", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &cfg); err != nil {
		t.Fatalf("生成的 YAML 不是合法配置: %v", err)
	}
	rules, ok := cfg["rules"].([]interface{})
	if !ok {
		t.Fatalf("生成的 YAML 缺少 rules")
	}

	for _, item := range rules {
		rule, _ := item.(string)
		if strings.Contains(rule, "wireguard-home") || strings.Contains(rule, "🏠 Home") {
			t.Fatalf("未生成对应 policy 时，不应保留规则，实际配置为:\n%s", config)
		}
	}
}

func TestBuildYAMLKeepsRulesForExistingPolicy(t *testing.T) {
	templateContent := `proxy-groups:
  - type: select
    name: 🏠 Home
    proxies: ["DIRECT", "__NODES__"]
proxies: []
rules:
  - RULE-SET,wireguard-home,🏠 Home
  - RULE-SET,need_direct,DIRECT
`

	nodes := []*ProxyNode{
		{
			Protocol: "wireguard",
			Name:     "🏠 wireguard-home",
			Server:   "vpn.example.com",
			Port:     51820,
			Options: map[string]interface{}{
				"ip":          "10.0.10.3/32",
				"private-key": "private-key",
				"public-key":  "peer-public-key",
				"allowed-ips": []interface{}{"0.0.0.0/0"},
			},
		},
	}

	config, err := BuildYAMLFromTemplateContent(nodes, templateContent, "stash")
	if err != nil {
		t.Fatalf("生成 Stash 配置失败: %v", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &cfg); err != nil {
		t.Fatalf("生成的 YAML 不是合法配置: %v", err)
	}
	rules, ok := cfg["rules"].([]interface{})
	if !ok {
		t.Fatalf("生成的 YAML 缺少 rules")
	}

	found := false
	for _, item := range rules {
		rule, _ := item.(string)
		if strings.Contains(rule, "RULE-SET,wireguard-home,🏠 Home") {
			found = true
		}
	}
	if !found {
		t.Fatalf("已存在对应 policy 时，应保留规则，实际配置为:\n%s", config)
	}
}

func TestBuildYAMLConfigForTargets(t *testing.T) {
	nodes := []*ProxyNode{
		{
			Protocol: "trojan",
			Name:     "TestNode",
			Server:   "test.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "test-password",
				"sni":      "test.example.com",
			},
		},
	}

	tests := []struct {
		name           string
		target         string
		expectedFields []string
		absentFields   []string
		wantErr        bool
	}{
		{
			name:   "Clash Meta 配置",
			target: "clash-meta",
			expectedFields: []string{
				"port", "socks-port", "external-controller", "proxies", "proxy-groups", "rules",
			},
			wantErr: false,
		},
		{
			name:   "Stash 配置",
			target: "stash",
			expectedFields: []string{
				"proxies", "proxy-groups", "rules",
			},
			absentFields: []string{
				"port", "socks-port", "external-controller", "tun",
			},
			wantErr: false,
		},
		{
			name:    "无效目标类型",
			target:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templatePath := getRuleTemplateFilePath()
			yamlData, err := buildYAMLFromSourceTemplate(nodes, templatePath, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildYAMLFromSourceTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			var cfg map[string]interface{}
			if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
				t.Fatalf("无法解析生成的 YAML: %v", err)
			}

			for _, field := range tt.expectedFields {
				if _, exists := cfg[field]; !exists {
					t.Errorf("配置中缺少期望的字段: %s", field)
				}
			}
			for _, field := range tt.absentFields {
				if _, exists := cfg[field]; exists {
					t.Errorf("配置中存在不应该存在的字段: %s", field)
				}
			}
			proxies, ok := cfg["proxies"].([]interface{})
			if !ok || len(proxies) != 1 {
				t.Errorf("期望有 1 个代理节点，实际有: %d", len(proxies))
			}
		})
	}
}

func TestFullConfigGeneration(t *testing.T) {
	templatePath := getRuleTemplateFilePath()
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("模板文件不存在，跳过测试")
	}

	nodes := []*ProxyNode{
		{
			Protocol: "trojan",
			Name:     "HK Node 1",
			Server:   "hk1.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "password123",
				"sni":      "hk1.example.com",
			},
		},
		{
			Protocol: "trojan",
			Name:     "US Node 1",
			Server:   "us1.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "password456",
				"sni":      "us1.example.com",
			},
		},
	}

	clashConfig, err := buildYAMLFromSourceTemplate(nodes, templatePath, "clash-meta")
	if err != nil {
		t.Fatalf("生成 Clash Meta 配置失败: %v", err)
	}
	if !strings.Contains(clashConfig, "port:") {
		t.Error("Clash Meta 配置缺少 port 字段")
	}
	if !strings.Contains(clashConfig, "HK Node 1") {
		t.Error("Clash Meta 配置缺少节点 HK Node 1")
	}

	stashConfig, err := buildYAMLFromSourceTemplate(nodes, templatePath, "stash")
	if err != nil {
		t.Fatalf("生成 Stash 配置失败: %v", err)
	}
	if strings.Contains(stashConfig, "port:") {
		var stashCfg map[string]interface{}
		if err := yaml.Unmarshal([]byte(stashConfig), &stashCfg); err == nil {
			if _, hasPort := stashCfg["port"]; hasPort {
				t.Error("Stash 配置不应包含 port 字段")
			}
		}
	}
	if strings.Contains(stashConfig, "external-controller:") {
		t.Error("Stash 配置不应包含 external-controller 字段")
	}
	if strings.Contains(stashConfig, "tun:") {
		t.Error("Stash 配置不应包含 tun 字段")
	}
	if !strings.Contains(stashConfig, "HK Node 1") {
		t.Error("Stash 配置缺少节点 HK Node 1")
	}
	if !strings.Contains(stashConfig, "proxies:") {
		t.Error("Stash 配置缺少 proxies 字段")
	}
	if !strings.Contains(stashConfig, "proxy-groups:") {
		t.Error("Stash 配置缺少 proxy-groups 字段")
	}
}

func TestAdaptConfigForTarget(t *testing.T) {
	cfg := map[string]interface{}{
		"port":                7890,
		"socks-port":          7891,
		"external-controller": "127.0.0.1:7892",
		"tun": map[string]interface{}{
			"enable": true,
		},
		"dns": map[string]interface{}{
			"enable":              true,
			"ipv6":                false,
			"prefer-h3":           true,
			"fake-ip-filter":      []string{"*"},
			"fake-ip-filter-mode": "blacklist",
		},
		"proxies": []interface{}{},
	}

	adaptConfigForTarget(cfg, "stash")

	if _, exists := cfg["port"]; exists {
		t.Error("Stash 配置不应包含 port 字段")
	}
	if _, exists := cfg["socks-port"]; exists {
		t.Error("Stash 配置不应包含 socks-port 字段")
	}
	if _, exists := cfg["external-controller"]; exists {
		t.Error("Stash 配置不应包含 external-controller 字段")
	}
	if _, exists := cfg["tun"]; exists {
		t.Error("Stash 配置不应包含 tun 字段")
	}

	dns, ok := cfg["dns"].(map[string]interface{})
	if !ok {
		t.Fatal("DNS 配置不存在或类型错误")
	}
	if _, exists := dns["prefer-h3"]; exists {
		t.Error("Stash DNS 配置不应包含 prefer-h3 字段")
	}
	if ipv6, exists := dns["ipv6"]; !exists || ipv6 != false {
		t.Error("Stash DNS 配置应保留 ipv6 字段")
	}
	if _, exists := dns["fake-ip-filter-mode"]; exists {
		t.Error("Stash DNS 配置不应包含 fake-ip-filter-mode 字段")
	}
	if _, exists := cfg["allow-lan"]; exists {
		t.Error("Stash 配置不应包含 allow-lan 字段")
	}
}
