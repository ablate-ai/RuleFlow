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
	Name                     string          `yaml:"name"`
	Type                     string          `yaml:"type"`
	Server                   string          `yaml:"server,omitempty"`
	Port                     int             `yaml:"port,omitempty"`
	Version                  int             `yaml:"version,omitempty"`
	Password                 string          `yaml:"password,omitempty"`
	UDP                      bool            `yaml:"udp,omitempty"`
	SNI                      string          `yaml:"sni,omitempty"`
	SkipCertVerify           bool            `yaml:"skip-cert-verify,omitempty"`
	Network                  string          `yaml:"network,omitempty"`
	Security                 string          `yaml:"security,omitempty"`
	UUID                     string          `yaml:"uuid,omitempty"`
	AlterID                  int             `yaml:"alterId,omitempty"`
	Cipher                   string          `yaml:"cipher,omitempty"`
	Flow                     string          `yaml:"flow,omitempty"`
	TLS                      bool            `yaml:"tls,omitempty"`
	Alpn                     []string        `yaml:"alpn,omitempty"`
	Fingerprint              string          `yaml:"client-fingerprint,omitempty"`
	Reality                  *RealityCfg     `yaml:"reality-opts,omitempty"`
	WSOpts                   *WSOpts         `yaml:"ws-opts,omitempty"`
	HTTPOpts                 *HTTPOpts       `yaml:"http-opts,omitempty"`
	GRPCOpts                 *GRPCOpts       `yaml:"grpc-opts,omitempty"`
	DialerProxy              string          `yaml:"dialer-proxy,omitempty"`
	IP                       string          `yaml:"ip,omitempty"`
	IPv6                     string          `yaml:"ipv6,omitempty"`
	PrivateKey               string          `yaml:"private-key,omitempty"`
	DNS                      []string        `yaml:"dns,omitempty"`
	MTU                      int             `yaml:"mtu,omitempty"`
	RemoteDNSResolve         bool            `yaml:"remote-dns-resolve,omitempty"`
	Peers                    []WireGuardPeer `yaml:"peers,omitempty"`
	IdleSessionCheckInterval int             `yaml:"idle-session-check-interval,omitempty"`
	IdleSessionTimeout       int             `yaml:"idle-session-timeout,omitempty"`
	MinIdleSession           int             `yaml:"min-idle-session,omitempty"`
}

type WSOpts struct {
	Path    string            `yaml:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type HTTPOpts struct {
	Path    string            `yaml:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type GRPCOpts struct {
	GrpcServiceName string `yaml:"grpc-service-name,omitempty"`
}

type RealityCfg struct {
	PublicKey string `yaml:"public-key,omitempty"`
	ShortID   string `yaml:"short-id,omitempty"`
}

type WireGuardPeer struct {
	Server              string   `yaml:"server,omitempty"`
	Port                int      `yaml:"port,omitempty"`
	PublicKey           string   `yaml:"public-key,omitempty"`
	PreSharedKey        string   `yaml:"pre-shared-key,omitempty"`
	AllowedIPs          []string `yaml:"allowed-ips,omitempty"`
	Reserved            []int    `yaml:"reserved,omitempty"`
	PersistentKeepalive int      `yaml:"persistent-keepalive,omitempty"`
}

type Group struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Proxies []string `yaml:"proxies"`
}

// ProxyNode 通用代理节点
type ProxyNode struct {
	Protocol    string                 // 协议类型: trojan, vmess, vless, ss, wireguard, anytls, hysteria2, tuic
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

type TLSOptions struct {
	Enabled    bool           `json:"enabled"`
	ServerName string         `json:"server_name,omitempty"`
	Insecure   bool           `json:"insecure,omitempty"`
	ALPN       []string       `json:"alpn,omitempty"`
	UTLS       *UTLSOptions   `json:"utls,omitempty"`
	Reality    *RealityConfig `json:"reality,omitempty"`
}

type UTLSOptions struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type TransportOptions struct {
	Type        string            `json:"type"`
	Path        string            `json:"path,omitempty"`
	Host        string            `json:"host,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ServiceName string            `json:"service_name,omitempty"`
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
		case "wireguard":
			addWireGuardFields(&proxy, node)
		case "anytls":
			addAnyTLSFields(&proxy, node.Options)
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
	applyTLSFields(proxy, opts)
	applyTransportFields(proxy, opts)
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
	applyTLSFields(proxy, opts)
	applyTransportFields(proxy, opts)
}

func addVLESSFields(proxy *Proxy, opts map[string]interface{}) {
	if uuid, ok := opts["uuid"].(string); ok {
		proxy.UUID = uuid
	}
	if flow, ok := opts["flow"].(string); ok {
		proxy.Flow = flow
	}
	applyTLSFields(proxy, opts)
	applyTransportFields(proxy, opts)
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

func addWireGuardFields(proxy *Proxy, node *ProxyNode) {
	cfg := extractWireGuardConfig(node)
	for _, addr := range cfg.LocalAddresses {
		if strings.Contains(addr, ":") {
			if proxy.IPv6 == "" {
				proxy.IPv6 = addr
			}
			continue
		}
		if proxy.IP == "" {
			proxy.IP = addr
		}
	}
	proxy.PrivateKey = cfg.PrivateKey
	proxy.DNS = append([]string(nil), cfg.DNS...)
	proxy.MTU = cfg.MTU
	proxy.RemoteDNSResolve = cfg.RemoteDNSResolve
	if len(cfg.Peers) > 0 {
		proxy.Server = ""
		proxy.Port = 0
		proxy.Peers = make([]WireGuardPeer, 0, len(cfg.Peers))
		for _, peer := range cfg.Peers {
			proxy.Peers = append(proxy.Peers, WireGuardPeer{
				Server:              peer.Server,
				Port:                peer.Port,
				PublicKey:           peer.PublicKey,
				PreSharedKey:        peer.PreSharedKey,
				AllowedIPs:          append([]string(nil), peer.AllowedIPs...),
				Reserved:            append([]int(nil), peer.Reserved...),
				PersistentKeepalive: peer.PersistentKeepalive,
			})
		}
	}
}

func addAnyTLSFields(proxy *Proxy, opts map[string]interface{}) {
	if password, ok := opts["password"].(string); ok {
		proxy.Password = password
	}
	if sni, ok := opts["sni"].(string); ok && sni != "" {
		proxy.SNI = sni
	}
	if skip, ok := boolOption(opts, "skipCertVerify", "skip-cert-verify"); ok {
		proxy.SkipCertVerify = skip
	}
	if fingerprint, ok := stringOption(opts, "fingerprint", "client-fingerprint"); ok {
		proxy.Fingerprint = fingerprint
	}
	if alpn, ok := stringSliceOption(opts, "alpn"); ok {
		proxy.Alpn = alpn
	}
	if v, ok := intOption(opts, "idleSessionCheckInterval", "idle-session-check-interval"); ok {
		proxy.IdleSessionCheckInterval = v
	}
	if v, ok := intOption(opts, "idleSessionTimeout", "idle-session-timeout"); ok {
		proxy.IdleSessionTimeout = v
	}
	if v, ok := intOption(opts, "minIdleSession", "min-idle-session"); ok {
		proxy.MinIdleSession = v
	}
}

func addTUICFields(proxy *Proxy, opts map[string]interface{}) {
	proxy.Version = 5
	if uuid, ok := opts["uuid"].(string); ok {
		proxy.UUID = uuid
	}
	if password, ok := opts["password"].(string); ok {
		proxy.Password = password
	}
	applyTLSFields(proxy, opts)
}

func applyTLSFields(proxy *Proxy, opts map[string]interface{}) {
	if tlsObj, ok := extractTLSOptions(opts); ok {
		proxy.TLS = tlsObj.Enabled
		if tlsObj.ServerName != "" {
			proxy.SNI = tlsObj.ServerName
		}
		proxy.SkipCertVerify = tlsObj.Insecure
		if len(tlsObj.ALPN) > 0 {
			proxy.Alpn = tlsObj.ALPN
		}
		if tlsObj.UTLS != nil && tlsObj.UTLS.Fingerprint != "" {
			proxy.Fingerprint = tlsObj.UTLS.Fingerprint
		}
		if tlsObj.Reality != nil && (tlsObj.Reality.PublicKey != "" || tlsObj.Reality.ShortID != "") {
			proxy.Reality = &RealityCfg{
				PublicKey: tlsObj.Reality.PublicKey,
				ShortID:   tlsObj.Reality.ShortID,
			}
		}
		if proxy.SNI == "" {
			if sni, ok := opts["sni"].(string); ok && sni != "" {
				proxy.SNI = sni
			}
		}
		if !proxy.SkipCertVerify {
			if skip, ok := boolOption(opts, "skipCertVerify", "skip-cert-verify"); ok {
				proxy.SkipCertVerify = skip
			}
		}
		if len(proxy.Alpn) == 0 {
			if alpn, ok := stringSliceOption(opts, "alpn"); ok {
				proxy.Alpn = alpn
			}
		}
		if proxy.Fingerprint == "" {
			if fingerprint, ok := stringOption(opts, "fingerprint", "client-fingerprint"); ok {
				proxy.Fingerprint = fingerprint
			}
		}
		if proxy.Reality == nil {
			if reality := extractLegacyRealityOptions(opts); reality != nil {
				proxy.Reality = reality
			}
		}
		return
	}

	if sni, ok := opts["sni"].(string); ok && sni != "" {
		proxy.SNI = sni
	}
	if skip, ok := boolOption(opts, "skipCertVerify", "skip-cert-verify"); ok {
		proxy.SkipCertVerify = skip
	}
	if tls, ok := boolOption(opts, "tls"); ok {
		proxy.TLS = tls
	}
	if alpn, ok := stringSliceOption(opts, "alpn"); ok {
		proxy.Alpn = alpn
	}
	if fingerprint, ok := stringOption(opts, "fingerprint", "client-fingerprint"); ok {
		proxy.Fingerprint = fingerprint
	}
	if proxy.Reality == nil {
		if reality := extractLegacyRealityOptions(opts); reality != nil {
			proxy.Reality = reality
		}
	}
}

func applyTransportFields(proxy *Proxy, opts map[string]interface{}) {
	if transport, ok := extractTransportOptions(opts); ok {
		if transport.Type != "" {
			proxy.Network = transport.Type
		}
		switch transport.Type {
		case "ws":
			proxy.WSOpts = &WSOpts{
				Path:    transport.Path,
				Headers: transport.Headers,
			}
		case "http", "httpupgrade":
			proxy.HTTPOpts = &HTTPOpts{
				Path:    transport.Path,
				Headers: transport.Headers,
			}
		case "grpc":
			if transport.ServiceName != "" {
				proxy.GRPCOpts = &GRPCOpts{GrpcServiceName: transport.ServiceName}
			}
		}
		return
	}

	if network, ok := opts["network"].(string); ok && network != "" {
		proxy.Network = network
	} else if ws, ok := opts["ws"].(bool); ok && ws {
		proxy.Network = "ws"
	}
	if wsPath, ok := opts["wsPath"].(string); ok && wsPath != "" {
		proxy.WSOpts = &WSOpts{Path: wsPath}
	} else if wsPath, ok := opts["ws-path"].(string); ok && wsPath != "" {
		proxy.WSOpts = &WSOpts{Path: wsPath}
	}
}

func extractTLSOptions(opts map[string]interface{}) (*TLSOptions, bool) {
	raw, exists := opts["tls"]
	if !exists || raw == nil {
		return nil, false
	}

	switch value := raw.(type) {
	case *TLSOptions:
		return value, true
	case map[string]interface{}:
		tlsObj := &TLSOptions{}
		if enabled, ok := boolOption(value, "enabled"); ok {
			tlsObj.Enabled = enabled
		}
		if serverName, ok := stringOption(value, "server_name", "server-name", "sni"); ok {
			tlsObj.ServerName = serverName
		}
		if insecure, ok := boolOption(value, "insecure", "skip-cert-verify", "skipCertVerify"); ok {
			tlsObj.Insecure = insecure
		}
		if alpn, ok := stringSliceOption(value, "alpn"); ok {
			tlsObj.ALPN = alpn
		}
		if utlsRaw, ok := value["utls"].(map[string]interface{}); ok {
			utlsObj := &UTLSOptions{}
			if enabled, ok := boolOption(utlsRaw, "enabled"); ok {
				utlsObj.Enabled = enabled
			}
			if fingerprint, ok := stringOption(utlsRaw, "fingerprint"); ok {
				utlsObj.Fingerprint = fingerprint
			}
			if utlsObj.Enabled || utlsObj.Fingerprint != "" {
				tlsObj.UTLS = utlsObj
			}
		}
		if realityRaw, ok := value["reality"].(map[string]interface{}); ok {
			realityObj := &RealityConfig{}
			realityObj.PublicKey, _ = stringOption(realityRaw, "public_key", "public-key", "PublicKey")
			realityObj.ShortID, _ = stringOption(realityRaw, "short_id", "short-id", "ShortID")
			if realityObj.PublicKey != "" || realityObj.ShortID != "" {
				tlsObj.Reality = realityObj
			}
		}
		return tlsObj, true
	case bool:
		return &TLSOptions{Enabled: value}, true
	default:
		return nil, false
	}
}

func extractTransportOptions(opts map[string]interface{}) (*TransportOptions, bool) {
	raw, exists := opts["transport"]
	if !exists || raw == nil {
		return nil, false
	}

	switch value := raw.(type) {
	case *TransportOptions:
		return value, true
	case map[string]interface{}:
		transport := &TransportOptions{}
		transport.Type, _ = stringOption(value, "type")
		transport.Path, _ = stringOption(value, "path")
		transport.Host, _ = stringOption(value, "host")
		transport.ServiceName, _ = stringOption(value, "service_name", "serviceName")
		if headersRaw, ok := value["headers"].(map[string]interface{}); ok {
			headers := make(map[string]string, len(headersRaw))
			for k, v := range headersRaw {
				if s, ok := v.(string); ok && s != "" {
					headers[k] = s
				}
			}
			if len(headers) > 0 {
				transport.Headers = headers
			}
		} else if headersRaw, ok := value["headers"].(map[string]string); ok && len(headersRaw) > 0 {
			headers := make(map[string]string, len(headersRaw))
			for k, v := range headersRaw {
				if v != "" {
					headers[k] = v
				}
			}
			if len(headers) > 0 {
				transport.Headers = headers
			}
		}
		return transport, true
	default:
		return nil, false
	}
}

func extractLegacyRealityOptions(opts map[string]interface{}) *RealityCfg {
	for _, key := range []string{"reality", "reality-opts"} {
		raw, exists := opts[key]
		if !exists || raw == nil {
			continue
		}
		switch r := raw.(type) {
		case *RealityConfig:
			if r.PublicKey != "" || r.ShortID != "" {
				return &RealityCfg{PublicKey: r.PublicKey, ShortID: r.ShortID}
			}
		case map[string]interface{}:
			pk, _ := stringOption(r, "public-key", "public_key", "PublicKey")
			sid, _ := stringOption(r, "short-id", "short_id", "ShortID")
			if pk != "" || sid != "" {
				return &RealityCfg{PublicKey: pk, ShortID: sid}
			}
		}
	}
	return nil
}

func stringOption(opts map[string]interface{}, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := opts[key].(string); ok && value != "" {
			return value, true
		}
	}
	return "", false
}

func boolOption(opts map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		if value, ok := opts[key].(bool); ok {
			return value, true
		}
	}
	return false, false
}

func intOption(opts map[string]interface{}, keys ...string) (int, bool) {
	for _, key := range keys {
		switch value := opts[key].(type) {
		case int:
			return value, true
		case int64:
			return int(value), true
		case float64:
			return int(value), true
		}
	}
	return 0, false
}

func stringSliceOption(opts map[string]interface{}, keys ...string) ([]string, bool) {
	for _, key := range keys {
		switch value := opts[key].(type) {
		case []string:
			if len(value) > 0 {
				return value, true
			}
		case []interface{}:
			out := make([]string, 0, len(value))
			for _, item := range value {
				if s, ok := item.(string); ok && s != "" {
					out = append(out, s)
				}
			}
			if len(out) > 0 {
				return out, true
			}
		}
	}
	return nil, false
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
