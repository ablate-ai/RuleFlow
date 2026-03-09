package app

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// 辅助函数：返回两个整数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 测试配置文件生成
func TestBuildYAMLConfigForTargets(t *testing.T) {
	// 创建测试节点
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
			name:   "Clash 配置",
			target: "clash",
			expectedFields: []string{
				"port", "socks-port", "external-controller", "proxies", "proxy-groups", "rules",
			},
			absentFields: []string{},
			wantErr:      false,
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
			name:           "无效目标类型",
			target:         "invalid",
			expectedFields: []string{},
			absentFields:   []string{},
			wantErr:        true,
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

			// 解析生成的 YAML
			var cfg map[string]interface{}
			if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
				t.Fatalf("无法解析生成的 YAML: %v", err)
			}

			// 检查期望存在的字段
			for _, field := range tt.expectedFields {
				if _, exists := cfg[field]; !exists {
					t.Errorf("配置中缺少期望的字段: %s", field)
				}
			}

			// 检查期望不存在的字段
			for _, field := range tt.absentFields {
				if _, exists := cfg[field]; exists {
					t.Errorf("配置中存在不应该存在的字段: %s", field)
				}
			}

			// 验证代理节点
			proxies, ok := cfg["proxies"].([]interface{})
			if !ok || len(proxies) != 1 {
				t.Errorf("期望有 1 个代理节点，实际有: %d", len(proxies))
			}
		})
	}
}

// 测试完整流程
func TestFullConfigGeneration(t *testing.T) {
	// 跳过测试如果模板文件不存在
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

	// 测试 Clash 配置
	clashConfig, err := buildYAMLFromSourceTemplate(nodes, templatePath, "clash")
	if err != nil {
		t.Fatalf("生成 Clash 配置失败: %v", err)
	}

	// 验证 Clash 配置包含必要字段
	if !strings.Contains(clashConfig, "port:") {
		t.Error("Clash 配置缺少 port 字段")
	}
	if !strings.Contains(clashConfig, "HK Node 1") {
		t.Error("Clash 配置缺少节点 HK Node 1")
	}

	// 测试 Stash 配置
	stashConfig, err := buildYAMLFromSourceTemplate(nodes, templatePath, "stash")
	if err != nil {
		t.Fatalf("生成 Stash 配置失败: %v", err)
	}

	// 验证 Stash 配置不包含 Clash 特定字段
	if strings.Contains(stashConfig, "port:") {
		// 检查是否是嵌套在其他地方的 port
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

	// 验证 Stash 配置包含必要字段
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

// 测试适配函数
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
			"prefer-h3":           true,
			"fake-ip-filter":      []string{"*"},
			"fake-ip-filter-mode": "blacklist",
		},
		"proxies": []interface{}{},
	}

	// 测试 Stash 适配
	adaptConfigForTarget(cfg, "stash")

	// 验证 Clash 特定字段被移除
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

	// 验证 DNS 配置被调整
	dns, ok := cfg["dns"].(map[string]interface{})
	if !ok {
		t.Fatal("DNS 配置不存在或类型错误")
	}

	if _, exists := dns["prefer-h3"]; exists {
		t.Error("Stash DNS 配置不应包含 prefer-h3 字段")
	}
	if _, exists := dns["fake-ip-filter-mode"]; exists {
		t.Error("Stash DNS 配置不应包含 fake-ip-filter-mode 字段")
	}

	// 验证 allow-lan 已被移除（Stash 不需要这个配置）
	if _, exists := cfg["allow-lan"]; exists {
		t.Error("Stash 配置不应包含 allow-lan 字段")
	}
}
