package app

import (
	"fmt"
	"strings"
)

// Clash 配置结构
type ClashConfig struct {
	Port            int      `yaml:"port"`
	SocksPort       int      `yaml:"socks-port"`
	RedirPort       int      `yaml:"redir-port"`
	MixedPort       int      `yaml:"mixed-port"`
	AllowLan        bool     `yaml:"allow-lan"`
	Mode            string   `yaml:"mode"`
	LogLevel        string   `yaml:"log-level"`
	ExternalControl string   `yaml:"external-controller"`
	Proxies         []Proxy  `yaml:"proxies"`
	ProxyGroups     []Group  `yaml:"proxy-groups"`
	Rules           []string `yaml:"rules"`
}

type Proxy struct {
	Name           string      `yaml:"name"`
	Type           string      `yaml:"type"`
	Server         string      `yaml:"server"`
	Port           int         `yaml:"port"`
	Password       string      `yaml:"password,omitempty"`
	UDP            bool        `yaml:"udp,omitempty"`
	SNI            string      `yaml:"sni,omitempty"`
	SkipCertVerify bool        `yaml:"skip-cert-verify,omitempty"`
	Network        string      `yaml:"network,omitempty"`
	Security       string      `yaml:"security,omitempty"`
	UUID           string      `yaml:"uuid,omitempty"`
	AlterID        int         `yaml:"alterId,omitempty"`
	Cipher         string      `yaml:"cipher,omitempty"`
	Flow           string      `yaml:"flow,omitempty"`
	TLS            bool        `yaml:"tls,omitempty"`
	Alpn           []string    `yaml:"alpn,omitempty"`
	Fingerprint    string      `yaml:"client-fingerprint,omitempty"`
	Reality        *RealityCfg `yaml:"reality-opts,omitempty"`
	WSOpts         *WSOpts     `yaml:"ws-opts,omitempty"`
	HTTPOpts       *HTTPOpts   `yaml:"http-opts,omitempty"`
	DialerProxy    string      `yaml:"dialer-proxy,omitempty"`
}

type WSOpts struct {
	Path    string            `yaml:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type HTTPOpts struct {
	Path    string            `yaml:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type RealityCfg struct {
	PublicKey string `yaml:"public-key,omitempty"`
	ShortID   string `yaml:"short-id,omitempty"`
}

type Group struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
}

// ProxyNode 通用代理节点
type ProxyNode struct {
	Protocol    string                 // 协议类型: trojan, vmess, vless, ss, ss2022, hysteria2, tuic
	Name        string                 // 节点名称
	Server      string                 // 服务器地址
	Port        int                    // 端口
	Options     map[string]interface{} // 协议特定选项
	DialerProxy string                 // 链式代理中转名称；Surge 导出时会映射为 underlying-proxy
}

// 各协议的选项结构
type TrojanOptions struct {
	Password       string
	SNI            string
	SkipCertVerify bool
}

type VMessOptions struct {
	UUID        string
	AlterID     int
	Security    string
	Network     string
	WSPath      string
	TLS         bool
	SNI         string
	Alpn        []string
	Fingerprint string
}

type VLESSOptions struct {
	UUID        string
	Flow        string
	Network     string
	WSPath      string
	TLS         bool
	SNI         string
	Fingerprint string
	Reality     *RealityConfig
}

type RealityConfig struct {
	PublicKey string `json:"public-key"`
	ShortID   string `json:"short-id"`
}

type ShadowsocksOptions struct {
	Cipher   string
	Password string
}

type Hysteria2Options struct {
	Password       string
	SNI            string
	SkipCertVerify bool
}

type TUICOptions struct {
	UUID     string
	Password string
	SNI      string
}

// Trojan 节点信息（保留以向后兼容）
type TrojanNode struct {
	Name     string
	Server   string
	Port     int
	Password string
	SNI      string
}

var defaultRules = []string{
	"DOMAIN-SUFFIX,local,DIRECT",
	"IP-CIDR,127.0.0.0/8,DIRECT",
	"IP-CIDR,172.16.0.0/12,DIRECT",
	"IP-CIDR,192.168.0.0/16,DIRECT",
	"IP-CIDR,10.0.0.0/8,DIRECT",
	"DOMAIN-SUFFIX,cn,DIRECT",
	"DOMAIN-KEYWORD,-cn,DIRECT",
	"GEOIP,CN,DIRECT",
	"MATCH,🚀 节点选择",
}

func cloneRules(rules []string) []string {
	return append([]string(nil), rules...)
}

// buildProxies 从通用节点构建代理配置
func buildProxies(nodes []*ProxyNode) ([]Proxy, []string) {
	proxies := make([]Proxy, 0, len(nodes))
	proxyNames := make([]string, 0, len(nodes))

	for i, node := range nodes {
		proxyName := ensureNodeName(node, i)
		proxyNames = append(proxyNames, proxyName)

		proxy := Proxy{
			Name:        proxyName,
			Type:        node.Protocol,
			Server:      node.Server,
			Port:        node.Port,
			UDP:         true,
			DialerProxy: node.DialerProxy,
		}

		// 根据协议类型添加特定字段
		switch node.Protocol {
		case "trojan":
			addTrojanFields(&proxy, node.Options)
		case "vmess":
			addVMessFields(&proxy, node.Options)
		case "vless":
			addVLESSFields(&proxy, node.Options)
		case "ss":
			addShadowsocksFields(&proxy, node.Options)
		case "hysteria2", "hy2":
			addHysteria2Fields(&proxy, node.Options)
		case "tuic":
			addTUICFields(&proxy, node.Options)
		}

		proxies = append(proxies, proxy)
	}

	return proxies, proxyNames
}

// buildProxiesFromTrojan 从 Trojan 节点构建代理配置（向后兼容）
func buildProxiesFromTrojan(nodes []TrojanNode) ([]Proxy, []string) {
	proxies := make([]Proxy, 0, len(nodes))
	proxyNames := make([]string, 0, len(nodes))

	for i, node := range nodes {
		proxyName := fmt.Sprintf("Trojan-%d", i+1)
		if node.Name != "" && node.Name != node.Server {
			proxyName = node.Name
		}

		proxies = append(proxies, Proxy{
			Name:           proxyName,
			Type:           "trojan",
			Server:         node.Server,
			Port:           node.Port,
			Password:       node.Password,
			UDP:            true,
			SNI:            node.SNI,
			SkipCertVerify: true,
		})
		proxyNames = append(proxyNames, proxyName)
	}

	return proxies, proxyNames
}

// ensureNodeName 确保节点有名称，并自动在前面添加国家 emoji
func ensureNodeName(node *ProxyNode, index int) string {
	var name string
	if node.Name != "" && node.Name != node.Server {
		name = node.Name
	} else {
		protocol := strings.ToUpper(node.Protocol)
		name = fmt.Sprintf("%s-%d", protocol, index+1)
	}
	return addCountryEmoji(name)
}

// 协议特定字段添加函数
func addTrojanFields(proxy *Proxy, opts map[string]interface{}) {
	if password, ok := opts["password"].(string); ok {
		proxy.Password = password
	}
	if sni, ok := opts["sni"].(string); ok && sni != "" {
		proxy.SNI = sni
	}
	if skip, ok := opts["skipCertVerify"].(bool); ok {
		proxy.SkipCertVerify = skip
	}
	if network, ok := opts["network"].(string); ok && network != "" {
		proxy.Network = network
	} else if ws, ok := opts["ws"].(bool); ok && ws {
		proxy.Network = "ws"
	}
	if wsPath, ok := opts["wsPath"].(string); ok && wsPath != "" {
		proxy.WSOpts = &WSOpts{
			Path: wsPath,
		}
	} else if wsPath, ok := opts["ws-path"].(string); ok && wsPath != "" {
		proxy.WSOpts = &WSOpts{
			Path: wsPath,
		}
	}
}

func addVMessFields(proxy *Proxy, opts map[string]interface{}) {
	if uuid, ok := opts["uuid"].(string); ok {
		proxy.UUID = uuid
	}
	if alterID, ok := opts["alterID"].(int); ok {
		proxy.AlterID = alterID
	}
	if security, ok := opts["security"].(string); ok {
		proxy.Security = security
	}
	if network, ok := opts["network"].(string); ok {
		proxy.Network = network
	}
	if tls, ok := opts["tls"].(bool); ok {
		proxy.TLS = tls
	}
	if sni, ok := opts["sni"].(string); ok {
		proxy.SNI = sni
	}
	if alpn, ok := opts["alpn"].([]string); ok {
		proxy.Alpn = alpn
	}
	if fingerprint, ok := opts["fingerprint"].(string); ok {
		proxy.Fingerprint = fingerprint
	}
	if wsPath, ok := opts["wsPath"].(string); ok && wsPath != "" {
		proxy.WSOpts = &WSOpts{
			Path: wsPath,
		}
	}
}

func addVLESSFields(proxy *Proxy, opts map[string]interface{}) {
	if uuid, ok := opts["uuid"].(string); ok {
		proxy.UUID = uuid
	}
	if flow, ok := opts["flow"].(string); ok {
		proxy.Flow = flow
	}
	if network, ok := opts["network"].(string); ok {
		proxy.Network = network
	}
	if tls, ok := opts["tls"].(bool); ok {
		proxy.TLS = tls
	}
	if sni, ok := opts["sni"].(string); ok {
		proxy.SNI = sni
	}
	// 兼容 URL 解析（"fingerprint"）和 Clash YAML 解析（"client-fingerprint"）两种 key
	if fingerprint, ok := opts["fingerprint"].(string); ok {
		proxy.Fingerprint = fingerprint
	} else if fingerprint, ok := opts["client-fingerprint"].(string); ok {
		proxy.Fingerprint = fingerprint
	}
	if wsPath, ok := opts["wsPath"].(string); ok && wsPath != "" {
		proxy.WSOpts = &WSOpts{
			Path: wsPath,
		}
	}
	// 兼容两种 key："reality"（vless:// URL 解析）和 "reality-opts"（Clash YAML 解析）
	// 同时兼容两种类型：*RealityConfig 和 map[string]interface{}
	for _, key := range []string{"reality", "reality-opts"} {
		raw, exists := opts[key]
		if !exists || raw == nil {
			continue
		}
		switch r := raw.(type) {
		case *RealityConfig:
			if r.PublicKey != "" || r.ShortID != "" {
				proxy.Reality = &RealityCfg{PublicKey: r.PublicKey, ShortID: r.ShortID}
			}
		case map[string]interface{}:
			// 兼容两种 key 格式：
			//   Clash YAML / 加 tag 后的 JSON: "public-key" / "short-id"
			//   旧 DB 数据（无 tag，PascalCase）: "PublicKey" / "ShortID"
			pk, _ := r["public-key"].(string)
			if pk == "" {
				pk, _ = r["PublicKey"].(string)
			}
			sid, _ := r["short-id"].(string)
			if sid == "" {
				sid, _ = r["ShortID"].(string)
			}
			if pk != "" || sid != "" {
				proxy.Reality = &RealityCfg{PublicKey: pk, ShortID: sid}
			}
		}
		break
	}
}

func addShadowsocksFields(proxy *Proxy, opts map[string]interface{}) {
	if cipher, ok := opts["cipher"].(string); ok {
		proxy.Cipher = cipher
	}
	if password, ok := opts["password"].(string); ok {
		proxy.Password = password
	}
	// SS 通常不需要 UDP
	proxy.UDP = false
}

func addHysteria2Fields(proxy *Proxy, opts map[string]interface{}) {
	if password, ok := opts["password"].(string); ok {
		proxy.Password = password
	}
	if sni, ok := opts["sni"].(string); ok && sni != "" {
		proxy.SNI = sni
	}
	if skip, ok := opts["skipCertVerify"].(bool); ok {
		proxy.SkipCertVerify = skip
	}
}

func addTUICFields(proxy *Proxy, opts map[string]interface{}) {
	if uuid, ok := opts["uuid"].(string); ok {
		proxy.UUID = uuid
	}
	if password, ok := opts["password"].(string); ok {
		proxy.Password = password
	}
	if sni, ok := opts["sni"].(string); ok && sni != "" {
		proxy.SNI = sni
	}
}

func builtInProxyName(name string) bool {
	switch name {
	case "DIRECT", "REJECT", "REJECT-DROP", "PASS", "COMPATIBLE":
		return true
	default:
		return false
	}
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
