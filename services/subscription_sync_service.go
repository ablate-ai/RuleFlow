package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/c.chen/ruleflow/database"
	"github.com/c.chen/ruleflow/internal/app"
)

// SubscriptionSyncService 订阅同步服务
type SubscriptionSyncService struct {
	subRepo  *database.SubscriptionRepo
	nodeRepo *database.NodeRepo
}

// NewSubscriptionSyncService 创建订阅同步服务
func NewSubscriptionSyncService(
	subRepo *database.SubscriptionRepo,
	nodeRepo *database.NodeRepo,
) *SubscriptionSyncService {
	return &SubscriptionSyncService{
		subRepo:  subRepo,
		nodeRepo: nodeRepo,
	}
}

// SyncSubscription 同步指定订阅的节点
// 使用完全替换策略：删除该订阅的旧节点，插入新节点
func (s *SubscriptionSyncService) SyncSubscription(ctx context.Context, subscriptionName string) (int, error) {
	log.Printf("[sync] 开始同步订阅: %s", subscriptionName)

	// 1. 获取订阅配置
	sub, err := s.subRepo.GetByName(ctx, subscriptionName)
	if err != nil {
		return 0, fmt.Errorf("订阅不存在: %s", subscriptionName)
	}

	if !sub.Enabled {
		return 0, fmt.Errorf("订阅已禁用: %s", subscriptionName)
	}

	if sub.URL == nil || *sub.URL == "" {
		return 0, fmt.Errorf("订阅没有配置 URL: %s", subscriptionName)
	}

	// 2. 从 URL 拉取订阅内容
	log.Printf("[sync] 拉取订阅内容: %s", *sub.URL)
	content, err := s.fetchSubscriptionContent(ctx, *sub.URL)
	if err != nil {
		log.Printf("[sync] 拉取失败: %v", err)
		_ = s.subRepo.UpdateFetchResult(ctx, subscriptionName, 0, err)
		return 0, fmt.Errorf("拉取订阅内容失败: %w", err)
	}
	log.Printf("[sync] 拉取成功，内容长度: %d 字节", len(content))

	// 3. 解析节点
	log.Printf("[sync] 开始解析节点...")
	nodes, err := app.ParseSubscription(content)
	if err != nil {
		log.Printf("[sync] 解析失败: %v", err)
		_ = s.subRepo.UpdateFetchResult(ctx, subscriptionName, 0, err)
		return 0, fmt.Errorf("解析订阅内容失败: %w", err)
	}

	if len(nodes) == 0 {
		log.Printf("[sync] 未解析到任何节点")
		_ = s.subRepo.UpdateFetchResult(ctx, subscriptionName, 0, fmt.Errorf("订阅中没有有效节点"))
		return 0, fmt.Errorf("订阅中没有有效节点")
	}
	log.Printf("[sync] 解析完成，共 %d 个节点", len(nodes))

	// 4. 开始事务处理
	// 使用完全替换策略
	source := fmt.Sprintf("subscription:%s", subscriptionName)

	// 4.1 删除该订阅的旧节点
	deleted, err := s.nodeRepo.DeleteBySource(ctx, source)
	if err != nil {
		return 0, fmt.Errorf("删除旧节点失败: %w", err)
	}
	log.Printf("[sync] 已删除旧节点: %d 个", deleted)

	// 4.2 将解析的节点转换为数据库模型
	dbNodes := s.convertToDBNodes(nodes, sub.ID, source)

	// 4.3 批量插入新节点
	if err := s.nodeRepo.BatchCreate(ctx, dbNodes); err != nil {
		return 0, fmt.Errorf("插入新节点失败: %w", err)
	}

	// 5. 更新订阅同步状态
	now := time.Now()
	nodeCount := len(dbNodes)
	_ = s.subRepo.UpdateFetchResult(ctx, subscriptionName, nodeCount, nil)

	if err := s.updateNodesLastSyncedAt(ctx, source, now); err != nil {
		_ = err
	}

	log.Printf("[sync] 同步完成: %s，节点数: %d", subscriptionName, nodeCount)
	return nodeCount, nil
}

// fetchSubscriptionContent 从 URL 拉取订阅内容
func (s *SubscriptionSyncService) fetchSubscriptionContent(ctx context.Context, url string) (string, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
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
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应内容失败: %w", err)
	}

	return string(content), nil
}

// convertToDBNodes 将解析的节点转换为数据库模型
func (s *SubscriptionSyncService) convertToDBNodes(nodes []*app.ProxyNode, sourceID int, source string) []database.Node {
	dbNodes := make([]database.Node, 0, len(nodes))

	for _, node := range nodes {
		dbNode := database.Node{
			Name:     node.Name,
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
func (s *SubscriptionSyncService) GetSyncStatus(ctx context.Context, subscriptionName string) (map[string]interface{}, error) {
	source := fmt.Sprintf("subscription:%s", subscriptionName)

	// 获取订阅信息
	sub, err := s.subRepo.GetByName(ctx, subscriptionName)
	if err != nil {
		return nil, fmt.Errorf("订阅不存在: %s", subscriptionName)
	}

	// 获取节点数量
	nodeCount, err := s.nodeRepo.CountBySource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("获取节点数量失败: %w", err)
	}

	status := map[string]interface{}{
		"subscription_name": subscriptionName,
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
		count, err := s.SyncSubscription(ctx, sub.Name)
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
