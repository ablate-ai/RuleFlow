package app

import (
	"testing"
)

func TestParseTrojanNode(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *ProxyNode
		wantErr bool
	}{
		{
			name: "标准 Trojan 链接",
			url:  "trojan://password@example.com:443?security=tls&sni=example.com#TestNode",
			want: &ProxyNode{
				Protocol: "trojan",
				Name:     "TestNode",
				Server:   "example.com",
				Port:     443,
				Options: map[string]interface{}{
					"password":       "password",
					"sni":            "example.com",
					"skipCertVerify": false,
				},
			},
			wantErr: false,
		},
		{
			name: "带 skipCertVerify 的 Trojan 链接",
			url:  "trojan://password@example.com?allowInsecure=1#InsecureNode",
			want: &ProxyNode{
				Protocol: "trojan",
				Name:     "InsecureNode",
				Server:   "example.com",
				Port:     443,
				Options: map[string]interface{}{
					"password":       "password",
					"sni":            "example.com",
					"skipCertVerify": true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTrojanNode(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTrojanNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Protocol != tt.want.Protocol {
					t.Errorf("parseTrojanNode() Protocol = %v, want %v", got.Protocol, tt.want.Protocol)
				}
				if got.Name != tt.want.Name {
					t.Errorf("parseTrojanNode() Name = %v, want %v", got.Name, tt.want.Name)
				}
				if got.Server != tt.want.Server {
					t.Errorf("parseTrojanNode() Server = %v, want %v", got.Server, tt.want.Server)
				}
				if got.Port != tt.want.Port {
					t.Errorf("parseTrojanNode() Port = %v, want %v", got.Port, tt.want.Port)
				}
			}
		})
	}
}

func TestParseVLESSNode(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *ProxyNode
		wantErr bool
	}{
		{
			name: "标准 VLESS 链接",
			url:  "vless://uuid@example.com:443?encryption=none&type=tcp&security=tls&sni=example.com#VLESSNode",
			want: &ProxyNode{
				Protocol: "vless",
				Name:     "VLESSNode",
				Server:   "example.com",
				Port:     443,
				Options: map[string]interface{}{
					"uuid":    "uuid",
					"network": "tcp",
					"tls":     true,
					"sni":     "example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "VLESS with WebSocket",
			url:  "vless://uuid@example.com:443?type=ws&path=/ws#VLESSWS",
			want: &ProxyNode{
				Protocol: "vless",
				Name:     "VLESSWS",
				Server:   "example.com",
				Port:     443,
				Options: map[string]interface{}{
					"uuid":    "uuid",
					"network": "ws",
					"tls":     false,
					"wsPath":  "/ws",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVLESSNode(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVLESSNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Protocol != tt.want.Protocol {
					t.Errorf("parseVLESSNode() Protocol = %v, want %v", got.Protocol, tt.want.Protocol)
				}
				if got.Server != tt.want.Server {
					t.Errorf("parseVLESSNode() Server = %v, want %v", got.Server, tt.want.Server)
				}
			}
		})
	}
}

func TestParseShadowsocksNode(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *ProxyNode
		wantErr bool
	}{
		{
			name: "SIP002 格式 Shadowsocks 链接",
			url:  "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@example.com:8388#SSNode",
			want: &ProxyNode{
				Protocol: "ss",
				Name:     "SSNode",
				Server:   "example.com",
				Port:     8388,
				Options: map[string]interface{}{
					"cipher":   "aes-256-gcm",
					"password": "password",
				},
			},
			wantErr: false,
		},
		{
			name: "空格分隔格式 Shadowsocks 链接",
			url:  "ss://aes-256-gcm:password@example.com:8388#SSNode2",
			want: &ProxyNode{
				Protocol: "ss",
				Name:     "SSNode2",
				Server:   "example.com",
				Port:     8388,
				Options: map[string]interface{}{
					"cipher":   "aes-256-gcm",
					"password": "password",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseShadowsocksNode(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseShadowsocksNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.want != nil {
				if got == nil {
					t.Errorf("parseShadowsocksNode() returned nil, want non-nil")
					return
				}
				if got.Protocol != tt.want.Protocol {
					t.Errorf("parseShadowsocksNode() Protocol = %v, want %v", got.Protocol, tt.want.Protocol)
				}
				if got.Server != tt.want.Server {
					t.Errorf("parseShadowsocksNode() Server = %v, want %v", got.Server, tt.want.Server)
				}
				if got.Port != tt.want.Port {
					t.Errorf("parseShadowsocksNode() Port = %v, want %v", got.Port, tt.want.Port)
				}
			}
		})
	}
}

func TestParseHysteria2Node(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *ProxyNode
		wantErr bool
	}{
		{
			name: "标准 Hysteria2 链接",
			url:  "hysteria2://password@example.com:443?sni=example.com&insecure=1#H2Node",
			want: &ProxyNode{
				Protocol: "hysteria2",
				Name:     "H2Node",
				Server:   "example.com",
				Port:     443,
				Options: map[string]interface{}{
					"password":       "password",
					"sni":            "example.com",
					"skipCertVerify": true,
				},
			},
			wantErr: false,
		},
		{
			name: "hy2 短格式",
			url:  "hy2://password@example.com:443#HY2Node",
			want: &ProxyNode{
				Protocol: "hysteria2",
				Name:     "HY2Node",
				Server:   "example.com",
				Port:     443,
				Options: map[string]interface{}{
					"password":       "password",
					"sni":            "",
					"skipCertVerify": false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHysteria2Node(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHysteria2Node() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Protocol != tt.want.Protocol {
					t.Errorf("parseHysteria2Node() Protocol = %v, want %v", got.Protocol, tt.want.Protocol)
				}
				if got.Name != tt.want.Name {
					t.Errorf("parseHysteria2Node() Name = %v, want %v", got.Name, tt.want.Name)
				}
				if got.Server != tt.want.Server {
					t.Errorf("parseHysteria2Node() Server = %v, want %v", got.Server, tt.want.Server)
				}
			}
		})
	}
}

func TestParseTUICNode(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *ProxyNode
		wantErr bool
	}{
		{
			name: "标准 TUIC 链接",
			url:  "tuic://uuid:password@example.com:443?sni=example.com#TUICNode",
			want: &ProxyNode{
				Protocol: "tuic",
				Name:     "TUICNode",
				Server:   "example.com",
				Port:     443,
				Options: map[string]interface{}{
					"uuid":     "uuid",
					"password": "password",
					"sni":      "example.com",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTUICNode(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTUICNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Protocol != tt.want.Protocol {
					t.Errorf("parseTUICNode() Protocol = %v, want %v", got.Protocol, tt.want.Protocol)
				}
				if got.Server != tt.want.Server {
					t.Errorf("parseTUICNode() Server = %v, want %v", got.Server, tt.want.Server)
				}
			}
		})
	}
}

func TestParseNodeURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		check   func(*testing.T, *ProxyNode)
	}{
		{
			name:    "Trojan",
			url:     "trojan://pass@example.com#TrojanNode",
			wantErr: false,
			check: func(t *testing.T, n *ProxyNode) {
				if n.Protocol != "trojan" {
					t.Errorf("Expected protocol trojan, got %s", n.Protocol)
				}
			},
		},
		{
			name:    "VLESS",
			url:     "vless://uuid@example.com#VLESSNode",
			wantErr: false,
			check: func(t *testing.T, n *ProxyNode) {
				if n.Protocol != "vless" {
					t.Errorf("Expected protocol vless, got %s", n.Protocol)
				}
			},
		},
		{
			name:    "Hysteria2",
			url:     "hysteria2://pass@example.com#H2Node",
			wantErr: false,
			check: func(t *testing.T, n *ProxyNode) {
				if n.Protocol != "hysteria2" {
					t.Errorf("Expected protocol hysteria2, got %s", n.Protocol)
				}
			},
		},
		{
			name:    "TUIC",
			url:     "tuic://uuid:pass@example.com#TUICNode",
			wantErr: false,
			check: func(t *testing.T, n *ProxyNode) {
				if n.Protocol != "tuic" {
					t.Errorf("Expected protocol tuic, got %s", n.Protocol)
				}
			},
		},
		{
			name:    "不支持的协议",
			url:     "unknown://test",
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNodeURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNodeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestParseSubscription(t *testing.T) {
	// 混合协议订阅测试
	mixedContent := `trojan://pass1@example.com#Trojan1
vless://uuid@example.com#VLESS1
hysteria2://pass2@example.com#H2_1
tuic://uuid:pass@example.com#TUIC1
`

	nodes, err := ParseSubscription(mixedContent)
	if err != nil {
		t.Fatalf("parseSubscription() error = %v", err)
	}

	if len(nodes) != 4 {
		t.Errorf("parseSubscription() returned %d nodes, want 4", len(nodes))
	}

	protocols := make(map[string]bool)
	for _, node := range nodes {
		protocols[node.Protocol] = true
	}

	expectedProtocols := []string{"trojan", "vless", "hysteria2", "tuic"}
	for _, proto := range expectedProtocols {
		if !protocols[proto] {
			t.Errorf("parseSubscription() missing protocol %s", proto)
		}
	}
}
