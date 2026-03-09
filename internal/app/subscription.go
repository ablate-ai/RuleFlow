package app

import (
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// fetchSubscriptionContent 从订阅地址获取原始内容
func fetchSubscriptionContent(subURL string) (string, error) {
	if subURL == "" {
		return "", fmt.Errorf("订阅地址不能为空")
	}

	req, err := http.NewRequest("GET", subURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "clash-verge/v1.3.8")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("重定向次数过多")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取订阅失败（网络错误）: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errorHint := ""
		if len(body) > 0 {
			preview := string(body)
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			errorHint = fmt.Sprintf("，响应内容: %s", preview)
		}
		return "", fmt.Errorf("订阅服务器返回错误（HTTP %d）%s", resp.StatusCode, errorHint)
	}

	content, err := readResponseBody(resp)
	if err != nil {
		return "", fmt.Errorf("读取订阅内容失败: %w", err)
	}

	if len(content) == 0 {
		return "", fmt.Errorf("订阅服务器返回了空内容")
	}

	return string(content), nil
}

// fetchSubscription 从订阅地址获取并解析节点链接（向后兼容）
func fetchSubscription(subURL string) ([]string, error) {
	content, err := fetchSubscriptionContent(subURL)
	if err != nil {
		return nil, err
	}

	return parseSubscriptionToURLs(content)
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	reader := io.Reader(resp.Body)
	var closeFn func() error

	switch encoding {
	case "", "identity":
		// 无压缩
	case "gzip", "x-gzip":
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("解析 gzip 响应失败: %w", err)
		}
		reader = gr
		closeFn = gr.Close
	case "deflate":
		zr, err := zlib.NewReader(resp.Body)
		if err == nil {
			reader = zr
			closeFn = zr.Close
			break
		}
		fr := flate.NewReader(resp.Body)
		reader = fr
		closeFn = fr.Close
	default:
		return nil, fmt.Errorf("不支持的响应压缩格式: %s", encoding)
	}

	if closeFn != nil {
		defer closeFn()
	}

	return io.ReadAll(reader)
}
