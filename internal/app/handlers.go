package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type SubscriptionConfigService interface {
	GetConfig(ctx context.Context, name, target string, fetchFunc func(string) (string, int, error)) (string, int, error)
}

// loadHTMLTemplate 从文件加载 HTML 模板，支持动态注入订阅地址。
func loadHTMLTemplate(subURL string) (string, error) {
	content, err := os.ReadFile(ResolveProjectPath("web/index.html"))
	if err != nil {
		return "", fmt.Errorf("读取模板文件失败: %w", err)
	}

	html := string(content)
	if subURL != "" {
		// 使用 strings.Replacer 一次性完成所有转义，避免创建多个中间字符串
		replacer := strings.NewReplacer("\\", "\\\\", "'", "\\'")
		escapedURL := replacer.Replace(subURL)
		html = strings.Replace(
			html,
			"const [subUrl, setSubUrl] = useState('');",
			fmt.Sprintf("const [subUrl, setSubUrl] = useState('%s');", escapedURL),
			1,
		)
	}

	return html, nil
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	subURL := r.URL.Query().Get("url")
	html, err := loadHTMLTemplate(subURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, html)
}

func buildYAMLConfig(subURL string, target string, templatePath string) (string, int, string, error) {
	if subURL == "" {
		return "", 0, "", fmt.Errorf("请提供订阅地址（url 参数）")
	}

	// 默认使用 clash，如果 target 参数无效
	if target == "" {
		target = "clash"
	}
	target = strings.ToLower(target)

	// 验证目标类型
	if target != "clash" && target != "stash" {
		return "", 0, "", fmt.Errorf("不支持的目标类型: %s (支持: clash, stash)", target)
	}

	// 处理模板路径
	if templatePath == "" {
		templatePath = getRuleTemplateFilePath()
	}

	// 获取订阅内容
	content, err := fetchSubscriptionContent(subURL)
	if err != nil {
		return "", 0, "", err
	}

	// 解析订阅内容（支持多协议）
	nodes, err := ParseSubscription(content)
	if err != nil {
		return "", 0, "", fmt.Errorf("解析订阅失败: %w", err)
	}

	if len(nodes) == 0 {
		return "", 0, "", fmt.Errorf("没有找到有效的节点")
	}

	yamlData, err := buildYAMLFromSourceTemplate(nodes, templatePath, target)
	if err != nil {
		return "", 0, "", err
	}

	return yamlData, len(nodes), target, nil
}

func SubHandler(w http.ResponseWriter, r *http.Request) {
	subURL := r.URL.Query().Get("url")
	target := r.URL.Query().Get("target")
	template := r.URL.Query().Get("template")

	yamlConfig, count, resolvedTemplate, err := buildYAMLConfig(subURL, target, template)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	// 根据目标类型设置不同的文件名
	filename := "clash_config.yaml"
	if resolvedTemplate == "stash" {
		filename = "stash_config.yaml"
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("X-Node-Count", fmt.Sprintf("%d", count))
	w.Header().Set("X-Rule-Template", resolvedTemplate)
	fmt.Fprint(w, yamlConfig)
}

// subHandlerWithName 处理基于订阅名称的请求
func SubHandlerWithName(w http.ResponseWriter, r *http.Request, service SubscriptionConfigService) {
	// 从 URL 路径中提取订阅名称
	// 假设路径格式为 /sub/{name}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "请提供订阅名称")
		return
	}
	name := parts[2]

	// 获取目标类型
	target := r.URL.Query().Get("target")
	if target == "" {
		target = "clash"
	}
	target = strings.ToLower(target)

	// 验证目标类型
	if target != "clash" && target != "stash" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "不支持的目标类型: %s (支持: clash, stash)", target)
		return
	}

	// 获取模板参数
	template := r.URL.Query().Get("template")

	// 定义获取函数
	fetchFunc := func(subURL string) (string, int, error) {
		yaml, count, _, err := buildYAMLConfig(subURL, target, template)
		return yaml, count, err
	}

	// 获取配置
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	yamlConfig, nodeCount, err := service.GetConfig(ctx, name, target, fetchFunc)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, err.Error())
		return
	}

	// 根据目标类型设置不同的文件名
	filename := "clash_config.yaml"
	if target == "stash" {
		filename = "stash_config.yaml"
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("X-Node-Count", fmt.Sprintf("%d", nodeCount))
	w.Header().Set("X-Rule-Template", target)
	fmt.Fprint(w, yamlConfig)
}
