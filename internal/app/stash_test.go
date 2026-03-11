package app

import (
	"encoding/json"
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
			name:   "Clash Meta 配置",
			target: "clash-meta",
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

	// 测试 Clash Meta 配置
	clashConfig, err := buildYAMLFromSourceTemplate(nodes, templatePath, "clash-meta")
	if err != nil {
		t.Fatalf("生成 Clash Meta 配置失败: %v", err)
	}

	// 验证 Clash Meta 配置包含必要字段
	if !strings.Contains(clashConfig, "port:") {
		t.Error("Clash Meta 配置缺少 port 字段")
	}
	if !strings.Contains(clashConfig, "HK Node 1") {
		t.Error("Clash Meta 配置缺少节点 HK Node 1")
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

// 测试 VLESS REALITY 节点的 YAML 输出字段名
func TestVLESSRealityYAMLOutput(t *testing.T) {
	proxy := &Proxy{
		Name:    "东京",
		Type:    "vless",
		Server:  "154.31.116.16",
		Port:    45478,
		UUID:    "700229f2-3709-4fc5-8d8e-ae1af6ed8d58",
		Flow:    "xtls-rprx-vision",
		Network: "tcp",
		TLS:     true,
		SNI:     "music.apple.com",
		Fingerprint: "random",
		Reality: &RealityCfg{
			PublicKey: "Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM",
			ShortID:   "0892831900b76d85",
		},
	}

	out, err := yaml.Marshal(proxy)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	yamlStr := string(out)

	// 必须输出 reality-opts，而不是 reality
	if !strings.Contains(yamlStr, "reality-opts:") {
		t.Errorf("YAML 输出缺少 reality-opts 字段，实际输出:\n%s", yamlStr)
	}
	if strings.Contains(yamlStr, "\nreality:") {
		t.Errorf("YAML 输出不应包含 reality: 字段（应为 reality-opts:），实际输出:\n%s", yamlStr)
	}
}

// 测试 addVLESSFields 能正确处理 Clash YAML 格式的 options（key 为 reality-opts / client-fingerprint）
func TestAddVLESSFieldsFromClashYAML(t *testing.T) {
	proxy := &Proxy{Name: "东京", Type: "vless", Server: "154.31.116.16", Port: 45478, UDP: true}
	opts := map[string]interface{}{
		"uuid":               "700229f2-3709-4fc5-8d8e-ae1af6ed8d58",
		"flow":               "xtls-rprx-vision",
		"network":            "tcp",
		"tls":                true,
		"sni":                "music.apple.com",
		"client-fingerprint": "random",
		"reality-opts": map[string]interface{}{
			"public-key": "Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM",
			"short-id":   "0892831900b76d85",
		},
	}
	addVLESSFields(proxy, opts)

	if proxy.Fingerprint != "random" {
		t.Errorf("client-fingerprint 未映射，got %q", proxy.Fingerprint)
	}
	if proxy.Reality == nil {
		t.Fatal("reality-opts 未映射，proxy.Reality 为 nil")
	}
	if proxy.Reality.PublicKey != "Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM" {
		t.Errorf("public-key 错误，got %q", proxy.Reality.PublicKey)
	}
	if proxy.Reality.ShortID != "0892831900b76d85" {
		t.Errorf("short-id 错误，got %q", proxy.Reality.ShortID)
	}
}

func TestAddTrojanFieldsWithWebSocketOptions(t *testing.T) {
	proxy := &Proxy{Name: "US WS", Type: "trojan", Server: "cdn.wwm.app", Port: 443, UDP: true}
	opts := map[string]interface{}{
		"password": "test-password",
		"sni":      "cdn.wwm.app",
		"network":  "ws",
		"wsPath":   "/test-path",
	}

	addTrojanFields(proxy, opts)

	if proxy.Network != "ws" {
		t.Fatalf("期望 trojan network=ws，实际为 %q", proxy.Network)
	}
	if proxy.WSOpts == nil {
		t.Fatal("期望 trojan 生成 ws-opts，实际为 nil")
	}
	if proxy.WSOpts.Path != "/test-path" {
		t.Fatalf("期望 ws path 为 /test-path，实际为 %q", proxy.WSOpts.Path)
	}
}

func TestAddTrojanFieldsWithNestedTransportHost(t *testing.T) {
	proxy := &Proxy{Name: "US WS", Type: "trojan", Server: "cdn.wwm.app", Port: 443, UDP: true}
	opts := map[string]interface{}{
		"password": "test-password",
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": "cdn.wwm.app",
			"alpn":        []string{"h2", "http/1.1"},
		},
		"transport": map[string]interface{}{
			"type": "ws",
			"path": "/test-path",
			"host": "edge.example.com",
			"headers": map[string]interface{}{
				"Host": "edge.example.com",
			},
		},
	}

	addTrojanFields(proxy, opts)

	if proxy.WSOpts == nil {
		t.Fatal("期望 trojan 生成 ws-opts，实际为 nil")
	}
	if proxy.WSOpts.Headers["Host"] != "edge.example.com" {
		t.Fatalf("期望 ws host 为 edge.example.com，实际为 %#v", proxy.WSOpts.Headers)
	}
	if len(proxy.Alpn) != 2 || proxy.Alpn[0] != "h2" {
		t.Fatalf("期望从嵌套 tls 读取 alpn，实际为 %#v", proxy.Alpn)
	}
}

func TestAddVLESSFieldsWithNestedGRPCTransport(t *testing.T) {
	proxy := &Proxy{Name: "东京", Type: "vless", Server: "154.31.116.16", Port: 45478, UDP: true}
	opts := map[string]interface{}{
		"uuid": "700229f2-3709-4fc5-8d8e-ae1af6ed8d58",
		"tls": map[string]interface{}{
			"enabled":     true,
			"server_name": "music.apple.com",
		},
		"transport": map[string]interface{}{
			"type":         "grpc",
			"service_name": "grpc-svc",
		},
	}
	addVLESSFields(proxy, opts)

	if proxy.Network != "grpc" {
		t.Fatalf("期望 network=grpc，实际为 %q", proxy.Network)
	}
	if proxy.GRPCOpts == nil || proxy.GRPCOpts.GrpcServiceName != "grpc-svc" {
		t.Fatalf("期望 grpc-opts.grpc-service-name=grpc-svc，实际为 %#v", proxy.GRPCOpts)
	}
}


// 模拟完整手动导入路径：vless:// URL → DB JSON 往返 → 生成 YAML
func TestVLESSManualImportFullPath(t *testing.T) {
	nodeURL := "vless://700229f2-3709-4fc5-8d8e-ae1af6ed8d58@154.31.116.16:45478?type=tcp&security=reality&pbk=Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM&fp=random&sni=music.apple.com&sid=0892831900b76d85&spx=%2F&flow=xtls-rprx-vision#%E4%B8%9C%E4%BA%AC"

	node, err := parseVLESSNode(nodeURL)
	if err != nil {
		t.Fatalf("parseVLESSNode error: %v", err)
	}

	// 模拟 DB JSON 往返（*RealityConfig 无 tag 时序列化为 PascalCase）
	jsonBytes, _ := json.Marshal(node.Options)
	var dbConfig map[string]interface{}
	json.Unmarshal(jsonBytes, &dbConfig)

	t.Logf("DB JSON: %s", jsonBytes)
	t.Logf("DB Config reality: %#v", dbConfig["reality"])

	proxyNodes := []*ProxyNode{{
		Protocol: node.Protocol,
		Name:     node.Name,
		Server:   node.Server,
		Port:     node.Port,
		Options:  dbConfig,
	}}

	proxies, _ := buildProxies(proxyNodes)
	p := proxies[0]

	if p.Reality == nil {
		t.Fatal("proxy.Reality 为 nil，DB 路径修复失败")
	}
	if p.Reality.PublicKey != "Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM" {
		t.Errorf("PublicKey 错误: %q", p.Reality.PublicKey)
	}
	if p.Reality.ShortID != "0892831900b76d85" {
		t.Errorf("ShortID 错误: %q", p.Reality.ShortID)
	}
	t.Logf("Reality: %+v", p.Reality)
}
func TestAddVLESSFieldsFromDB(t *testing.T) {
	proxy := &Proxy{Name: "东京", Type: "vless", Server: "154.31.116.16", Port: 45478}
	opts := map[string]interface{}{
		"uuid":    "700229f2-3709-4fc5-8d8e-ae1af6ed8d58",
		"network": "tcp",
		"tls":     true,
		"sni":     "music.apple.com",
		"flow":    "xtls-rprx-vision",
		// DB JSON 反序列化：*RealityConfig{} 无 tag 时序列化为 PascalCase
		"reality": map[string]interface{}{
			"PublicKey": "Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM",
			"ShortID":   "0892831900b76d85",
		},
	}
	addVLESSFields(proxy, opts)

	if proxy.Reality == nil {
		t.Fatal("DB 路径: proxy.Reality 为 nil")
	}
	if proxy.Reality.PublicKey != "Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM" {
		t.Errorf("PublicKey 错误: %q", proxy.Reality.PublicKey)
	}
	if proxy.Reality.ShortID != "0892831900b76d85" {
		t.Errorf("ShortID 错误: %q", proxy.Reality.ShortID)
	}
}

func TestBuildProxiesSupportsAnyTLS(t *testing.T) {
	nodes := []*ProxyNode{
		{
			Protocol: "anytls",
			Name:     "SG AnyTLS",
			Server:   "sg.example.com",
			Port:     443,
			Options: map[string]interface{}{
				"password":            "secret",
				"sni":                 "sg.example.com",
				"client-fingerprint":  "chrome",
				"skip-cert-verify":    true,
				"alpn":                []interface{}{"h2", "http/1.1"},
				"idle-session-timeout": 30,
			},
		},
	}

	proxies, _ := buildProxies(nodes)
	if len(proxies) != 1 {
		t.Fatalf("buildProxies() 返回代理数 = %d, want 1", len(proxies))
	}

	proxy := proxies[0]
	if proxy.Type != "anytls" {
		t.Fatalf("proxy.Type = %s, want anytls", proxy.Type)
	}
	if proxy.Password != "secret" {
		t.Fatalf("proxy.Password = %s, want secret", proxy.Password)
	}
	if proxy.Fingerprint != "chrome" {
		t.Fatalf("proxy.Fingerprint = %s, want chrome", proxy.Fingerprint)
	}
	if !proxy.SkipCertVerify {
		t.Fatal("proxy.SkipCertVerify = false, want true")
	}
	if proxy.IdleSessionTimeout != 30 {
		t.Fatalf("proxy.IdleSessionTimeout = %d, want 30", proxy.IdleSessionTimeout)
	}
	if len(proxy.Alpn) != 2 || proxy.Alpn[0] != "h2" || proxy.Alpn[1] != "http/1.1" {
		t.Fatalf("proxy.Alpn = %#v, want [h2 http/1.1]", proxy.Alpn)
	}
}

func TestVLESSRealityEndToEnd(t *testing.T) {
	clashYAML := `proxies:
  - name: 东京
    type: vless
    server: 154.31.116.16
    port: 45478
    uuid: 700229f2-3709-4fc5-8d8e-ae1af6ed8d58
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    sni: music.apple.com
    client-fingerprint: random
    reality-opts:
      public-key: Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM
      short-id: "0892831900b76d85"
`
	nodes, err := parseClashYAML(clashYAML)
	if err != nil || len(nodes) == 0 {
		t.Fatalf("parseClashYAML failed: %v", err)
	}

	node := nodes[0]
	t.Logf("options map: %#v", node.Options)

	ro := node.Options["reality-opts"]
	t.Logf("reality-opts type=%T value=%#v", ro, ro)

	proxies, _ := buildProxies(nodes)
	if len(proxies) == 0 {
		t.Fatal("buildProxies returned empty")
	}
	p := proxies[0]
	if p.Reality == nil {
		t.Fatal("proxy.Reality is nil")
	}
	if p.Reality.PublicKey == "" {
		t.Errorf("PublicKey 为空，options=%#v", node.Options)
	}
	t.Logf("Reality=%+v", p.Reality)
}
