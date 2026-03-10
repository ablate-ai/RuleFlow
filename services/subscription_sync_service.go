package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c.chen/ruleflow/cache"
	"github.com/c.chen/ruleflow/database"
	"github.com/c.chen/ruleflow/internal/app"
)

// SubscriptionSyncService 订阅同步服务
type SubscriptionSyncService struct {
	subRepo  *database.SubscriptionRepo
	nodeRepo *database.NodeRepo
	subCache *cache.SubscriptionCache
}

// NewSubscriptionSyncService 创建订阅同步服务
func NewSubscriptionSyncService(
	subRepo *database.SubscriptionRepo,
	nodeRepo *database.NodeRepo,
	subCache *cache.SubscriptionCache,
) *SubscriptionSyncService {
	return &SubscriptionSyncService{
		subRepo:  subRepo,
		nodeRepo: nodeRepo,
		subCache: subCache,
	}
}

// SyncSubscription 同步指定订阅的节点
// 使用完全替换策略：删除该订阅的旧节点，插入新节点
func (s *SubscriptionSyncService) SyncSubscription(ctx context.Context, subscriptionID int) (int, error) {
	log.Printf("[sync] 开始同步订阅: id=%d", subscriptionID)

	// 1. 获取订阅配置
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("订阅不存在: %d", subscriptionID)
	}

	if !sub.Enabled {
		return 0, fmt.Errorf("订阅已禁用: %s", sub.Name)
	}

	if sub.URL == nil || *sub.URL == "" {
		return 0, fmt.Errorf("订阅没有配置 URL: %s", sub.Name)
	}

	// 2. 从 URL 拉取订阅内容
	log.Printf("[sync] 拉取订阅内容: %s", *sub.URL)
	content, userInfoHeader, err := s.fetchSubscriptionContent(ctx, *sub.URL)
	if err != nil {
		log.Printf("[sync] 拉取失败: %v", err)
		_ = s.subRepo.UpdateFetchResultByID(ctx, subscriptionID, 0, err)
		return 0, fmt.Errorf("拉取订阅内容失败: %w", err)
	}
	log.Printf("[sync] 拉取成功，内容长度: %d 字节", len(content))

	// 3. 解析节点
	log.Printf("[sync] 开始解析节点...")
	nodes, err := app.ParseSubscription(content)
	if err != nil {
		log.Printf("[sync] 解析失败: %v", err)
		_ = s.subRepo.UpdateFetchResultByID(ctx, subscriptionID, 0, err)
		return 0, fmt.Errorf("解析订阅内容失败: %w", err)
	}

	if len(nodes) == 0 {
		log.Printf("[sync] 未解析到任何节点")
		_ = s.subRepo.UpdateFetchResultByID(ctx, subscriptionID, 0, fmt.Errorf("订阅中没有有效节点"))
		return 0, fmt.Errorf("订阅中没有有效节点")
	}
	log.Printf("[sync] 解析完成，共 %d 个节点", len(nodes))

	// 4. 开始事务处理
	// 使用完全替换策略
	source := fmt.Sprintf("subscription:%s", sub.Name)
	namePrefix := fmt.Sprintf("[%s] ", strings.TrimSpace(sub.Name))

	// 4.1 删除该订阅的旧节点。按 source_id 删除，避免订阅改名后旧 source 残留。
	deleted, err := s.nodeRepo.DeleteBySourceID(ctx, sub.ID)
	if err != nil {
		return 0, fmt.Errorf("删除旧节点失败: %w", err)
	}
	log.Printf("[sync] 已删除旧节点: %d 个", deleted)

	// 4.2 将解析的节点转换为数据库模型
	dbNodes := s.convertToDBNodes(nodes, sub.ID, source, namePrefix)

	// 4.3 批量插入新节点
	if err := s.nodeRepo.BatchCreate(ctx, dbNodes); err != nil {
		return 0, fmt.Errorf("插入新节点失败: %w", err)
	}

	// 5. 更新订阅同步状态
	now := time.Now()
	nodeCount := len(dbNodes)
	_ = s.subRepo.UpdateFetchResultByID(ctx, subscriptionID, nodeCount, nil)

	if err := s.updateNodesLastSyncedAt(ctx, source, now); err != nil {
		_ = err
	}

	// 6. 解析并缓存流量信息
	if userInfoHeader != "" && s.subCache != nil {
		if info := parseUserInfo(userInfoHeader); info != nil {
			if err := s.subCache.SetUserInfo(ctx, sub.Name, info); err != nil {
				log.Printf("[sync] 缓存流量信息失败: %v", err)
			} else {
				log.Printf("[sync] 流量信息已缓存: upload=%d download=%d total=%d", info.Upload, info.Download, info.Total)
			}
		}
	}

	log.Printf("[sync] 同步完成: %s，节点数: %d", sub.Name, nodeCount)
	return nodeCount, nil
}

// parseUserInfo 解析 Subscription-Userinfo 响应头
// 格式：upload=1234; download=5678; total=10000000000; expire=1750000000
func parseUserInfo(header string) *cache.UserInfo {
	info := &cache.UserInfo{}
	found := false
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "upload":
			info.Upload = val
			found = true
		case "download":
			info.Download = val
			found = true
		case "total":
			info.Total = val
			found = true
		case "expire":
			info.Expire = &val
		}
	}
	if !found {
		return nil
	}
	return info
}

// fetchSubscriptionContent 从 URL 拉取订阅内容，同时返回 Subscription-Userinfo 响应头
func (s *SubscriptionSyncService) fetchSubscriptionContent(ctx context.Context, url string) (content string, userInfoHeader string, err error) {
	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Clash")
	req.Header.Set("Accept", "*/*")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应内容失败: %w", err)
	}

	return string(body), resp.Header.Get("Subscription-Userinfo"), nil
}

// convertToDBNodes 将解析的节点转换为数据库模型
func (s *SubscriptionSyncService) convertToDBNodes(nodes []*app.ProxyNode, sourceID int, source string, namePrefix string) []database.Node {
	dbNodes := make([]database.Node, 0, len(nodes))

	for _, node := range nodes {
		name := node.Name
		if namePrefix != "" {
			name = namePrefix + name
		}
		dbNode := database.Node{
			Name:     name,
			Protocol: node.Protocol,
			Server:   node.Server,
			Port:     node.Port,
			Config:   node.Options,
			Source:   source,
			SourceID: &sourceID,
			Enabled:  true,
			Tags:     []string{},
		}
		dbNodes = append(dbNodes, dbNode)
	}

	return dbNodes
}

// updateNodesLastSyncedAt 更新节点的最后同步时间
func (s *SubscriptionSyncService) updateNodesLastSyncedAt(ctx context.Context, source string, syncTime time.Time) error {
	// 这个方法可以扩展为批量更新操作
	// 目前在插入时已经设置了 created_at，这里可以添加一个专门的方法
	// 暂时保留为空实现，因为 created_at 可以作为同步时间的参考
	return nil
}

// GetSyncStatus 获取订阅同步状态
func (s *SubscriptionSyncService) GetSyncStatus(ctx context.Context, subscriptionID int) (map[string]interface{}, error) {
	// 获取订阅信息
	sub, err := s.subRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("订阅不存在: %d", subscriptionID)
	}
	// 获取节点数量
	nodeCount, err := s.nodeRepo.CountBySourceID(ctx, sub.ID)
	if err != nil {
		return nil, fmt.Errorf("获取节点数量失败: %w", err)
	}

	status := map[string]interface{}{
		"subscription_id":   sub.ID,
		"subscription_name": sub.Name,
		"enabled":           sub.Enabled,
		"node_count":        nodeCount,
		"last_fetched_at":   sub.LastFetchedAt,
		"last_fetch_error":  sub.LastFetchError,
	}

	return status, nil
}

// SyncAllSubscriptions 同步所有启用的订阅
func (s *SubscriptionSyncService) SyncAllSubscriptions(ctx context.Context) (map[string]int, error) {
	// 获取所有订阅
	subs, err := s.subRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取订阅列表失败: %w", err)
	}

	results := make(map[string]int)
	successCount := 0
	failureCount := 0

	for _, sub := range subs {
		if !sub.Enabled {
			continue
		}

		// 同步订阅
		count, err := s.SyncSubscription(ctx, sub.ID)
		if err != nil {
			failureCount++
			results[sub.Name] = 0
			continue
		}

		successCount++
		results[sub.Name] = count
	}

	return results, nil
}
