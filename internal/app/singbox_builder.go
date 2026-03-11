package app

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func getSingBoxTemplateFilePath() string {
	return ResolveProjectPath("rules/sing-box.json.template")
}

func BuildSingBoxFromDefaultTemplate(nodes []*ProxyNode) (string, error) {
	data, err := os.ReadFile(getSingBoxTemplateFilePath())
	if err != nil {
		return "", fmt.Errorf("读取 sing-box 模板文件失败: %w", err)
	}
	return BuildSingBoxFromTemplateContent(nodes, string(data))
}

func BuildSingBoxFromTemplateContent(nodes []*ProxyNode, templateContent string) (string, error) {
	nodeOutbounds, nodeEndpoints, nodeNames := buildSingBoxOutbounds(nodes)
	allNodes := fallbackSingBoxOutboundList(nodeNames, []string{"DIRECT"})
	autoNodes := fallbackSingBoxOutboundList(filterNodeNames(nodeNames, func(name string) bool {
		return !strings.Contains(strings.ToLower(name), "v2")
	}), allNodes)
	sgNodes := fallbackSingBoxOutboundList(filterNodeNames(nodeNames, regionMatcher("新加坡", "SG", "Singapore", "狮城", "🇸🇬")), allNodes)
	hkNodes := fallbackSingBoxOutboundList(filterNodeNames(nodeNames, regionMatcher("香港", "HK", "Hong Kong", "🇭🇰")), allNodes)
	usNodes := fallbackSingBoxOutboundList(filterNodeNames(nodeNames, regionMatcher("美国", "US", "USA", "United States", "🇺🇸")), allNodes)
	jpNodes := fallbackSingBoxOutboundList(filterNodeNames(nodeNames, regionMatcher("日本", "JP", "Japan", "🇯🇵")), allNodes)
	aiPrimaryNodes := fallbackSingBoxOutboundList(usNodes, allNodes)

	extraOutbounds := make([]map[string]interface{}, 0, len(nodeOutbounds)+1)
	extraOutbounds = append(extraOutbounds, map[string]interface{}{
		"type":                        "selector",
		"tag":                         "🇺🇸 AI-PRIMARY",
		"outbounds":                   aiPrimaryNodes,
		"default":                     aiPrimaryNodes[0],
		"interrupt_exist_connections": true,
	})
	extraOutbounds = append(extraOutbounds, nodeOutbounds...)

	replacer := strings.NewReplacer(
		"\"__ENDPOINTS__\"", marshalSingBoxRawObjectArray(nodeEndpoints),
		"__ENDPOINTS__", marshalSingBoxRawObjectArray(nodeEndpoints),
		"\"__PROXY_NODES__\"", joinSingBoxStringLiterals(allNodes),
		"\"__AUTO_NODES__\"", joinSingBoxStringLiterals(autoNodes),
		"\"__SG_NODES__\"", joinSingBoxStringLiterals(sgNodes),
		"\"__HK_NODES__\"", joinSingBoxStringLiterals(hkNodes),
		"\"__US_NODES__\"", joinSingBoxStringLiterals(usNodes),
		"\"__JP_NODES__\"", joinSingBoxStringLiterals(jpNodes),
		"\"__OUTBOUNDS__\"", joinSingBoxRawObjects(extraOutbounds),
	)

	content := replacer.Replace(templateContent)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return "", fmt.Errorf("生成 sing-box 配置失败: %w", err)
	}
	if err := expandSingBoxOutboundGroups(parsed, nodeNames, map[string][]string{
		"__NODES__":       allNodes,
		"__PROXY_NODES__": allNodes,
		"__AUTO_NODES__":  autoNodes,
		"__SG_NODES__":    sgNodes,
		"__HK_NODES__":    hkNodes,
		"__US_NODES__":    usNodes,
		"__JP_NODES__":    jpNodes,
	}); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("格式化 sing-box 配置失败: %w", err)
	}
	return string(formatted), nil
}

func buildSingBoxOutbounds(nodes []*ProxyNode) ([]map[string]interface{}, []map[string]interface{}, []string) {
	outbounds := make([]map[string]interface{}, 0, len(nodes))
	endpoints := make([]map[string]interface{}, 0)
	names := make([]string, 0, len(nodes))

	for i, node := range nodes {
		name := ensureNodeName(node, i)
		if strings.EqualFold(node.Protocol, "wireguard") {
			endpoints = append(endpoints, singBoxWireGuardEndpoint(node, name))
		} else {
			outbounds = append(outbounds, singBoxOutbound(node, name))
		}
		names = append(names, name)
	}

	return outbounds, endpoints, names
}

func singBoxOutbound(node *ProxyNode, tag string) map[string]interface{} {
	outboundType := normalizeSingBoxProtocol(node.Protocol)
	outbound := map[string]interface{}{
		"type": outboundType,
		"tag":  tag,
	}
	if outboundType != "wireguard" {
		outbound["server"] = node.Server
		outbound["server_port"] = node.Port
	}

	if node.DialerProxy != "" {
		outbound["detour"] = addCountryEmoji(node.DialerProxy)
	}

	switch outboundType {
	case "trojan":
		if password, ok := stringOption(node.Options, "password"); ok {
			outbound["password"] = password
		}
		outbound["tls"] = singBoxTLSObject(node.Options, node.Server, true)
		if transport := singBoxTransportObject(node.Options); transport != nil {
			outbound["transport"] = transport
		}
	case "vmess":
		if uuid, ok := stringOption(node.Options, "uuid"); ok {
			outbound["uuid"] = uuid
		}
		if security, ok := stringOption(node.Options, "security"); ok {
			outbound["security"] = security
		}
		if alterID, ok := intOption(node.Options, "alterID"); ok {
			outbound["alter_id"] = alterID
		}
		if tlsObj := singBoxTLSObject(node.Options, node.Server, false); tlsObj != nil {
			outbound["tls"] = tlsObj
		}
		if transport := singBoxTransportObject(node.Options); transport != nil {
			outbound["transport"] = transport
		}
	case "vless":
		if uuid, ok := stringOption(node.Options, "uuid"); ok {
			outbound["uuid"] = uuid
		}
		if flow, ok := stringOption(node.Options, "flow"); ok {
			outbound["flow"] = flow
		}
		if tlsObj := singBoxTLSObject(node.Options, node.Server, false); tlsObj != nil {
			outbound["tls"] = tlsObj
		}
		if transport := singBoxTransportObject(node.Options); transport != nil {
			outbound["transport"] = transport
		}
	case "shadowsocks":
		if method, ok := stringOption(node.Options, "cipher", "method"); ok {
			outbound["method"] = method
		}
		if password, ok := stringOption(node.Options, "password"); ok {
			outbound["password"] = password
		}
	case "hysteria2":
		if password, ok := stringOption(node.Options, "password"); ok {
			outbound["password"] = password
		}
		outbound["tls"] = singBoxTLSObject(node.Options, node.Server, true)
	case "anytls":
		if password, ok := stringOption(node.Options, "password"); ok {
			outbound["password"] = password
		}
		outbound["tls"] = singBoxTLSObject(node.Options, node.Server, true)
		if v, ok := intOption(node.Options, "idleSessionCheckInterval", "idle-session-check-interval"); ok {
			outbound["idle_session_check_interval"] = fmt.Sprintf("%ds", v)
		}
		if v, ok := intOption(node.Options, "idleSessionTimeout", "idle-session-timeout"); ok {
			outbound["idle_session_timeout"] = fmt.Sprintf("%ds", v)
		}
		if v, ok := intOption(node.Options, "minIdleSession", "min-idle-session"); ok {
			outbound["min_idle_session"] = v
		}
	case "tuic":
		if uuid, ok := stringOption(node.Options, "uuid"); ok {
			outbound["uuid"] = uuid
		}
		if password, ok := stringOption(node.Options, "password"); ok {
			outbound["password"] = password
		}
		outbound["tls"] = singBoxTLSObject(node.Options, node.Server, true)
	case "wireguard":
		cfg := extractWireGuardConfig(node)
		if len(cfg.LocalAddresses) > 0 {
			outbound["local_address"] = append([]string(nil), cfg.LocalAddresses...)
		}
		if cfg.PrivateKey != "" {
			outbound["private_key"] = cfg.PrivateKey
		}
		if len(cfg.Peers) > 1 {
			peers := make([]map[string]interface{}, 0, len(cfg.Peers))
			for _, peer := range cfg.Peers {
				item := map[string]interface{}{
					"server":      peer.Server,
					"server_port": peer.Port,
					"public_key":  peer.PublicKey,
				}
				if len(peer.AllowedIPs) > 0 {
					item["allowed_ips"] = append([]string(nil), peer.AllowedIPs...)
				}
				if peer.PreSharedKey != "" {
					item["pre_shared_key"] = peer.PreSharedKey
				}
				if len(peer.Reserved) > 0 {
					item["reserved"] = append([]int(nil), peer.Reserved...)
				}
				if peer.PersistentKeepalive > 0 {
					item["persistent_keepalive_interval"] = peer.PersistentKeepalive
				}
				peers = append(peers, item)
			}
			outbound["peers"] = peers
		} else if len(cfg.Peers) == 1 {
			peer := cfg.Peers[0]
			outbound["server"] = peer.Server
			outbound["server_port"] = peer.Port
			outbound["peer_public_key"] = peer.PublicKey
			if len(peer.AllowedIPs) > 0 {
				outbound["peer_allowed_ips"] = append([]string(nil), peer.AllowedIPs...)
			}
			if peer.PreSharedKey != "" {
				outbound["pre_shared_key"] = peer.PreSharedKey
			}
			if len(peer.Reserved) > 0 {
				outbound["reserved"] = append([]int(nil), peer.Reserved...)
			}
		}
		if cfg.MTU > 0 {
			outbound["mtu"] = cfg.MTU
		}
	}

	return outbound
}

func singBoxWireGuardEndpoint(node *ProxyNode, tag string) map[string]interface{} {
	cfg := extractWireGuardConfig(node)
	endpoint := map[string]interface{}{
		"type": "wireguard",
		"tag":  tag,
	}
	if len(cfg.LocalAddresses) > 0 {
		endpoint["address"] = append([]string(nil), cfg.LocalAddresses...)
	}
	if cfg.PrivateKey != "" {
		endpoint["private_key"] = cfg.PrivateKey
	}
	if cfg.MTU > 0 {
		endpoint["mtu"] = cfg.MTU
	}
	if len(cfg.Peers) > 0 {
		peers := make([]map[string]interface{}, 0, len(cfg.Peers))
		for _, peer := range cfg.Peers {
			item := map[string]interface{}{
				"address":    peer.Server,
				"port":       peer.Port,
				"public_key": peer.PublicKey,
			}
			if len(peer.AllowedIPs) > 0 {
				item["allowed_ips"] = append([]string(nil), peer.AllowedIPs...)
			}
			if peer.PreSharedKey != "" {
				item["pre_shared_key"] = peer.PreSharedKey
			}
			if len(peer.Reserved) > 0 {
				item["reserved"] = append([]int(nil), peer.Reserved...)
			}
			if peer.PersistentKeepalive > 0 {
				item["persistent_keepalive_interval"] = peer.PersistentKeepalive
			}
			peers = append(peers, item)
		}
		endpoint["peers"] = peers
	}
	return endpoint
}

func normalizeSingBoxProtocol(protocol string) string {
	switch strings.ToLower(protocol) {
	case "ss":
		return "shadowsocks"
	case "hy2", "hysteria2":
		return "hysteria2"
	case "wireguard":
		return "wireguard"
	default:
		return strings.ToLower(protocol)
	}
}

func singBoxTransportObject(opts map[string]interface{}) map[string]interface{} {
	if transport, ok := extractTransportOptions(opts); ok && transport != nil && transport.Type != "" {
		obj := map[string]interface{}{
			"type": transport.Type,
		}
		if transport.Path != "" {
			obj["path"] = transport.Path
		}
		if transport.Host != "" {
			obj["host"] = transport.Host
		}
		if len(transport.Headers) > 0 {
			obj["headers"] = transport.Headers
		}
		if transport.ServiceName != "" {
			obj["service_name"] = transport.ServiceName
		}
		return obj
	}
	return nil
}

func singBoxTLSObject(opts map[string]interface{}, server string, force bool) map[string]interface{} {
	if tlsOptions, ok := extractTLSOptions(opts); ok && tlsOptions != nil {
		if !force && !tlsOptions.Enabled && tlsOptions.ServerName == "" && !tlsOptions.Insecure && len(tlsOptions.ALPN) == 0 && tlsOptions.UTLS == nil && tlsOptions.Reality == nil {
			return nil
		}

		tlsObj := map[string]interface{}{
			"enabled": force || tlsOptions.Enabled || tlsOptions.Reality != nil,
		}
		if tlsOptions.ServerName != "" {
			tlsObj["server_name"] = tlsOptions.ServerName
		} else if force {
			tlsObj["server_name"] = server
		}
		if tlsOptions.Insecure {
			tlsObj["insecure"] = tlsOptions.Insecure
		}
		if len(tlsOptions.ALPN) > 0 {
			tlsObj["alpn"] = tlsOptions.ALPN
		}
		if tlsOptions.UTLS != nil && tlsOptions.UTLS.Fingerprint != "" {
			tlsObj["utls"] = map[string]interface{}{
				"enabled":     tlsOptions.UTLS.Enabled || tlsOptions.UTLS.Fingerprint != "",
				"fingerprint": tlsOptions.UTLS.Fingerprint,
			}
		}
		if tlsOptions.Reality != nil && (tlsOptions.Reality.PublicKey != "" || tlsOptions.Reality.ShortID != "") {
			realityObj := map[string]interface{}{
				"enabled": true,
			}
			if tlsOptions.Reality.PublicKey != "" {
				realityObj["public_key"] = tlsOptions.Reality.PublicKey
			}
			if tlsOptions.Reality.ShortID != "" {
				realityObj["short_id"] = tlsOptions.Reality.ShortID
			}
			tlsObj["reality"] = realityObj
		}
		return tlsObj
	}
	return nil
}

func joinSingBoxStringLiterals(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		b, _ := json.Marshal(value)
		parts = append(parts, string(b))
	}
	return strings.Join(parts, ", ")
}

func joinSingBoxRawObjects(values []map[string]interface{}) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		b, _ := json.Marshal(value)
		parts = append(parts, string(b))
	}
	return strings.Join(parts, ",\n    ")
}

func marshalSingBoxRawObjectArray(values []map[string]interface{}) string {
	if len(values) == 0 {
		return "[]"
	}
	return "[\n    " + joinSingBoxRawObjects(values) + "\n  ]"
}

func fallbackSingBoxOutboundList(values []string, fallback []string) []string {
	values = dedupeStrings(values)
	if len(values) == 0 {
		return append([]string(nil), fallback...)
	}
	return values
}

func expandSingBoxOutboundGroups(root map[string]interface{}, nodeNames []string, placeholders map[string][]string) error {
	rawOutbounds, ok := root["outbounds"].([]interface{})
	if !ok {
		return nil
	}

	known := make(map[string]struct{}, len(nodeNames)+len(rawOutbounds)+len(placeholders))
	for _, name := range nodeNames {
		known[name] = struct{}{}
	}
	for _, item := range rawOutbounds {
		outbound, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if tag, ok := outbound["tag"].(string); ok && tag != "" {
			known[tag] = struct{}{}
		}
	}

	for i, item := range rawOutbounds {
		outbound, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		rawMembers, ok := outbound["outbounds"].([]interface{})
		if !ok {
			continue
		}

		filterPattern, _ := outbound["filter"].(string)
		delete(outbound, "filter")
		var filterRe *regexp.Regexp
		if filterPattern != "" {
			filterRe, _ = regexp.Compile(filterPattern)
		}

		filterNames := func(names []string) []string {
			if filterRe == nil {
				return append([]string(nil), names...)
			}
			filtered := make([]string, 0, len(names))
			for _, name := range names {
				if filterRe.MatchString(name) {
					filtered = append(filtered, name)
				}
			}
			return filtered
		}

		expanded := make([]string, 0, len(rawMembers))
		for _, member := range rawMembers {
			name, ok := member.(string)
			if !ok || strings.TrimSpace(name) == "" {
				continue
			}

			if names, exists := placeholders[name]; exists {
				expanded = append(expanded, filterNames(names)...)
				continue
			}
			if _, exists := known[name]; exists || builtInProxyName(name) {
				expanded = append(expanded, name)
			}
		}

		if len(expanded) == 0 {
			switch strings.ToLower(strings.TrimSpace(fmt.Sprint(outbound["type"]))) {
			case "selector", "urltest", "url-test", "fallback", "load_balance", "load-balance":
				candidates := filterNames(placeholders["__NODES__"])
				if len(candidates) > 0 {
					expanded = append(expanded, candidates...)
				} else {
					expanded = append(expanded, "DIRECT")
				}
			default:
				expanded = append(expanded, "DIRECT")
			}
		}

		expanded = dedupeStrings(expanded)
		converted := make([]interface{}, 0, len(expanded))
		for _, name := range expanded {
			converted = append(converted, name)
		}
		outbound["outbounds"] = converted
		rawOutbounds[i] = outbound
	}

	root["outbounds"] = rawOutbounds
	return nil
}

func filterNodeNames(values []string, match func(string) bool) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if match(value) {
			out = append(out, value)
		}
	}
	return out
}

func regionMatcher(keywords ...string) func(string) bool {
	patterns := make([]*regexp.Regexp, 0, len(keywords))
	for _, keyword := range keywords {
		patterns = append(patterns, regexp.MustCompile("(?i)"+regexp.QuoteMeta(keyword)))
	}
	return func(name string) bool {
		for _, pattern := range patterns {
			if pattern.MatchString(name) {
				return true
			}
		}
		return false
	}
}
