package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildSingBoxFromTemplateContent(t *testing.T) {
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
		{
			Protocol: "vless",
			Name:     "HK Node 1",
			Server:   "hk1.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"uuid": "11111111-1111-1111-1111-111111111111",
				"tls":  true,
				"sni":  "hk1.example.com",
			},
		},
	}

	config, err := BuildSingBoxFromDefaultTemplate(nodes)
	if err != nil {
		t.Fatalf("生成 sing-box 配置失败: %v", err)
	}

	if strings.Contains(config, "__OUTBOUNDS__") || strings.Contains(config, "__AUTO_NODES__") {
		t.Fatalf("生成后的配置不应保留节点占位符，实际配置为:\n%s", config)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		t.Fatalf("生成的 sing-box 配置不是合法 JSON: %v", err)
	}

	outbounds, ok := cfg["outbounds"].([]interface{})
	if !ok || len(outbounds) == 0 {
		t.Fatalf("生成的 sing-box 配置缺少 outbounds")
	}

	foundUS := false
	foundHK := false
	for _, item := range outbounds {
		outbound, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tag, _ := outbound["tag"].(string)
		if strings.Contains(tag, "US Node 1") {
			foundUS = true
		}
		if strings.Contains(tag, "HK Node 1") {
			foundHK = true
		}
	}

	if !foundUS || !foundHK {
		t.Fatalf("生成的 sing-box 配置缺少节点出站，US=%v HK=%v", foundUS, foundHK)
	}
}

func TestBuildSingBoxTrojanWebSocket(t *testing.T) {
	nodes := []*ProxyNode{
		{
			Protocol: "trojan",
			Name:     "WS Node",
			Server:   "ws.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password": "password123",
				"sni":      "ws.example.com",
				"network":  "ws",
				"wsPath":   "/ws",
			},
		},
	}

	config, err := BuildSingBoxFromDefaultTemplate(nodes)
	if err != nil {
		t.Fatalf("生成 sing-box 配置失败: %v", err)
	}

	if !strings.Contains(config, "\"type\": \"ws\"") || !strings.Contains(config, "\"path\": \"/ws\"") {
		t.Fatalf("期望生成 WebSocket transport，实际配置为:\n%s", config)
	}
}
