package app

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildSurgeFromTemplateContent 从模板内容（字符串）构建 Surge INI 配置
func BuildSurgeFromTemplateContent(nodes []*ProxyNode, templateContent string) (string, error) {
	// 收集节点名称和节点行
	nodeNames := make([]string, 0, len(nodes))
	nodeLines := make([]string, 0, len(nodes))
	for i, node := range nodes {
		name := ensureNodeName(node, i)
		nodeNames = append(nodeNames, name)
		nodeLines = append(nodeLines, surgeProxyLine(node, name))
	}

	lines := strings.Split(templateContent, "\n")
	var out []string

	// 当前所在的 section
	section := ""
	// 用于 [Proxy] section：记录是否已找到 __NODES__ 占位行
	proxyNodesInserted := false
	// 缓存 [Proxy] section 末尾插入位置
	pendingProxyLines := []string(nil)

	filterRe := regexp.MustCompile(`(?i),?\s*filter=([^,\r\n]+)`)

	for idx, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		// 识别 section 头
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// 离开 [Proxy] section 时，若未找到 __NODES__ 则在末尾追加节点
			if section == "[Proxy]" && !proxyNodesInserted {
				out = append(out, pendingProxyLines...)
				proxyNodesInserted = true
			}
			pendingProxyLines = nil

			section = trimmed
			out = append(out, line)
			continue
		}

		_ = idx // 消除 unused 警告

		switch section {
		case "[Proxy]":
			if strings.TrimSpace(line) == "__NODES__" {
				// 替换占位符为所有节点行
				out = append(out, nodeLines...)
				proxyNodesInserted = true
			} else {
				// 暂存非占位行，以便在 section 结束时判断是否需要追加
				if !proxyNodesInserted {
					pendingProxyLines = append(pendingProxyLines, line)
				}
				out = append(out, line)
			}

		case "[Proxy Group]":
			if !strings.Contains(line, "__NODES__") {
				out = append(out, line)
				continue
			}

			// 提取 filter= 参数
			var filterPattern string
			match := filterRe.FindStringSubmatch(line)
			if match != nil {
				filterPattern = strings.TrimSpace(match[1])
				// 从行中移除 filter= 参数
				line = filterRe.ReplaceAllString(line, "")
			}

			// 按 filter 筛选节点
			filtered := filterNodesByPattern(nodeNames, filterPattern)

			// 替换 __NODES__ 为逗号分隔的节点名
			expansion := strings.Join(filtered, ", ")
			line = strings.ReplaceAll(line, "__NODES__", expansion)
			// 清理多余的逗号（如 __NODES__ 为空时可能产生 ",,"）
			line = cleanCommas(line)
			out = append(out, line)

		default:
			out = append(out, line)
		}
	}

	// 文件末尾若仍在 [Proxy] section 且未插入节点
	if section == "[Proxy]" && !proxyNodesInserted {
		out = append(out, nodeLines...)
	}

	return strings.Join(out, "\n"), nil
}

// BuildSurgeFromDefaultTemplate 使用内置默认结构生成最小 Surge 配置
func BuildSurgeFromDefaultTemplate(nodes []*ProxyNode) (string, error) {
	nodeNames := make([]string, 0, len(nodes))
	nodeLines := make([]string, 0, len(nodes))
	for i, node := range nodes {
		name := ensureNodeName(node, i)
		nodeNames = append(nodeNames, name)
		nodeLines = append(nodeLines, surgeProxyLine(node, name))
	}

	var sb strings.Builder

	sb.WriteString("[General]\n")
	sb.WriteString("loglevel = notify\n")
	sb.WriteString("dns-server = 8.8.8.8, 114.114.114.114\n")
	sb.WriteString("\n")

	sb.WriteString("[Proxy]\n")
	for _, l := range nodeLines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	sb.WriteString("[Proxy Group]\n")
	selectProxies := append([]string{"♻️ 自动选择", "DIRECT"}, nodeNames...)
	sb.WriteString(fmt.Sprintf("🚀 节点选择 = select, %s\n", strings.Join(selectProxies, ", ")))
	sb.WriteString(fmt.Sprintf("♻️ 自动选择 = url-test, %s, url=http://cp.cloudflare.com/generate_204, interval=300\n",
		strings.Join(nodeNames, ", ")))
	sb.WriteString("\n")

	sb.WriteString("[Rule]\n")
	sb.WriteString("DOMAIN-SUFFIX,local,DIRECT\n")
	sb.WriteString("IP-CIDR,127.0.0.0/8,DIRECT\n")
	sb.WriteString("IP-CIDR,172.16.0.0/12,DIRECT\n")
	sb.WriteString("IP-CIDR,192.168.0.0/16,DIRECT\n")
	sb.WriteString("IP-CIDR,10.0.0.0/8,DIRECT\n")
	sb.WriteString("DOMAIN-SUFFIX,cn,DIRECT\n")
	sb.WriteString("GEOIP,CN,DIRECT\n")
	sb.WriteString("FINAL,🚀 节点选择\n")

	return sb.String(), nil
}

// surgeProxyLine 将 ProxyNode 转换为 Surge 代理行
func surgeProxyLine(node *ProxyNode, name string) string {
	opts := node.Options
	if opts == nil {
		opts = map[string]interface{}{}
	}

	strOpt := func(key string) string {
		v, _ := opts[key].(string)
		return v
	}
	boolOpt := func(key string) bool {
		v, _ := opts[key].(bool)
		return v
	}

	switch node.Protocol {
	case "trojan":
		password := strOpt("password")
		sni := strOpt("sni")
		skipVerify := boolOpt("skipCertVerify")
		return fmt.Sprintf("%s = trojan, %s, %d, password=%s, sni=%s, skip-cert-verify=%v",
			name, node.Server, node.Port, password, sni, skipVerify)

	case "vmess":
		uuid := strOpt("uuid")
		security := strOpt("security")
		if security == "" {
			security = "auto"
		}
		tls := boolOpt("tls")
		sni := strOpt("sni")
		return fmt.Sprintf("%s = vmess, %s, %d, username=%s, encrypt-method=%s, tls=%v, sni=%s",
			name, node.Server, node.Port, uuid, security, tls, sni)

	case "vless":
		uuid := strOpt("uuid")
		tls := boolOpt("tls")
		sni := strOpt("sni")
		return fmt.Sprintf("%s = vless, %s, %d, uuid=%s, tls=%v, sni=%s",
			name, node.Server, node.Port, uuid, tls, sni)

	case "ss":
		cipher := strOpt("cipher")
		password := strOpt("password")
		return fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s",
			name, node.Server, node.Port, cipher, password)

	case "hysteria2", "hy2":
		password := strOpt("password")
		sni := strOpt("sni")
		skipVerify := boolOpt("skipCertVerify")
		return fmt.Sprintf("%s = hysteria2, %s, %d, password=%s, sni=%s, skip-cert-verify=%v",
			name, node.Server, node.Port, password, sni, skipVerify)

	case "tuic":
		uuid := strOpt("uuid")
		password := strOpt("password")
		sni := strOpt("sni")
		return fmt.Sprintf("%s = tuic-v5, %s, %d, token=%s, password=%s, sni=%s",
			name, node.Server, node.Port, uuid, password, sni)

	default:
		// 未知协议，生成注释行
		return fmt.Sprintf("# 不支持的协议: %s (%s)", node.Protocol, name)
	}
}

// filterNodesByPattern 按正则模式过滤节点名列表
func filterNodesByPattern(nodeNames []string, pattern string) []string {
	if pattern == "" {
		return nodeNames
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nodeNames
	}
	result := make([]string, 0, len(nodeNames))
	for _, n := range nodeNames {
		if re.MatchString(n) {
			result = append(result, n)
		}
	}
	return result
}

// cleanCommas 清理多余的逗号（如连续逗号或行尾逗号）
func cleanCommas(s string) string {
	// 替换连续逗号
	multiComma := regexp.MustCompile(`,\s*,`)
	for multiComma.MatchString(s) {
		s = multiComma.ReplaceAllString(s, ",")
	}
	// 去掉尾随逗号（在可能的空格后）
	s = regexp.MustCompile(`,\s*$`).ReplaceAllString(s, "")
	return s
}
