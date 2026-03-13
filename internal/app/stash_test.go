package app

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStashTUICUsesVersion5(t *testing.T) {
	templatePath := getRuleTemplateFilePath()
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("模板文件不存在，跳过测试")
	}

	nodes := []*ProxyNode{
		{
			Protocol: "tuic",
			Name:     "JP TUIC",
			Server:   "jp.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"uuid":     "949862c6-2854-475d-bf74-c73eda541d22",
				"password": "949862c6-2854-475d-bf74-c73eda541d22",
				"sni":      "jp.example.com",
			},
		},
	}

	stashConfig, err := buildYAMLFromSourceTemplate(nodes, templatePath, "stash")
	if err != nil {
		t.Fatalf("生成 Stash 配置失败: %v", err)
	}

	var cfg struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(stashConfig), &cfg); err != nil {
		t.Fatalf("解析 Stash 配置失败: %v", err)
	}
	if len(cfg.Proxies) != 1 {
		t.Fatalf("期望 1 个代理节点，实际为 %d", len(cfg.Proxies))
	}

	proxy := cfg.Proxies[0]
	if proxy["type"] != "tuic" {
		t.Fatalf("期望代理类型为 tuic，实际为 %v", proxy["type"])
	}
	if proxy["version"] != 5 {
		t.Fatalf("期望 TUIC 显式输出 version: 5，实际为 %v", proxy["version"])
	}
	if proxy["uuid"] != "949862c6-2854-475d-bf74-c73eda541d22" {
		t.Fatalf("期望保留 uuid，实际为 %v", proxy["uuid"])
	}
	if proxy["password"] != "949862c6-2854-475d-bf74-c73eda541d22" {
		t.Fatalf("期望保留 password，实际为 %v", proxy["password"])
	}
}

func TestStashWireGuardFlattensPeerFields(t *testing.T) {
	templatePath := getRuleTemplateFilePath()
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("模板文件不存在，跳过测试")
	}

	nodes := []*ProxyNode{
		{
			Protocol: "wireguard",
			Name:     "wireguard-home",
			Options: map[string]interface{}{
				"ip":          "10.0.10.3/32",
				"private-key": "private-key",
				"dns":         []string{"192.168.100.1"},
				"mtu":         1420,
				"peers": []interface{}{
					map[string]interface{}{
						"server":      "home.zzfzzf.com",
						"port":        51820,
						"public-key":  "peer-public-key",
						"allowed-ips": []interface{}{"192.168.100.0/24", "10.0.10.0/24"},
					},
				},
			},
		},
	}

	stashConfig, err := buildYAMLFromSourceTemplate(nodes, templatePath, "stash")
	if err != nil {
		t.Fatalf("生成 Stash 配置失败: %v", err)
	}

	var cfg struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(stashConfig), &cfg); err != nil {
		t.Fatalf("解析 Stash 配置失败: %v", err)
	}
	if len(cfg.Proxies) != 1 {
		t.Fatalf("期望 1 个代理节点，实际为 %d", len(cfg.Proxies))
	}

	proxy := cfg.Proxies[0]
	if proxy["type"] != "wireguard" {
		t.Fatalf("期望代理类型为 wireguard，实际为 %v", proxy["type"])
	}
	if proxy["server"] != "home.zzfzzf.com" {
		t.Fatalf("期望 server 为 home.zzfzzf.com，实际为 %v", proxy["server"])
	}
	if proxy["port"] != 51820 {
		t.Fatalf("期望 port 为 51820，实际为 %v", proxy["port"])
	}
	if proxy["public-key"] != "peer-public-key" {
		t.Fatalf("期望 public-key 为 peer-public-key，实际为 %v", proxy["public-key"])
	}
	if _, exists := proxy["peers"]; exists {
		t.Fatalf("Stash wireguard 不应输出 peers 字段，实际为 %v", proxy["peers"])
	}
}
