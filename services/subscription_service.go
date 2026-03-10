package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c.chen/ruleflow/cache"
	"github.com/c.chen/ruleflow/database"
)

// SubscriptionService 订阅服务
type SubscriptionService struct {
	repo  *database.SubscriptionRepo
	cache *cache.SubscriptionCache
}

// NewSubscriptionService 创建订阅服务
func NewSubscriptionService(repo *database.SubscriptionRepo, cache *cache.SubscriptionCache) *SubscriptionService {
	return &SubscriptionService{
		repo:  repo,
		cache: cache,
	}
}

// GetConfig 获取订阅配置
func (s *SubscriptionService) GetConfig(ctx context.Context, name, target string, fetchFunc func(string) (string, int, error)) (string, int, error) {
	return s.fetchConfig(ctx, name, fetchFunc)
}

// fetchConfig 从上游获取配置
func (s *SubscriptionService) fetchConfig(ctx context.Context, name string, fetchFunc func(string) (string, int, error)) (string, int, error) {
	// 从数据库获取订阅配置
	sub, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return "", 0, fmt.Errorf("订阅不存在: %s", name)
	}

	if !sub.Enabled {
		return "", 0, fmt.Errorf("订阅已禁用: %s", name)
	}

	// URL 订阅
	if sub.URL == nil || strings.TrimSpace(*sub.URL) == "" {
		return "", 0, fmt.Errorf("订阅缺少 URL: %s", name)
	}

	// 调用获取函数获取配置
	yaml, nodeCount, err := fetchFunc(*sub.URL)
	if err != nil {
		// 更新数据库记录错误
		_ = s.repo.UpdateFetchResult(ctx, name, 0, err)
		return "", 0, fmt.Errorf("获取订阅配置失败: %w", err)
	}

	// 更新数据库记录成功
	_ = s.repo.UpdateFetchResult(ctx, name, nodeCount, nil)

	return yaml, nodeCount, nil
}

// RefreshConfig 手动刷新订阅配置
func (s *SubscriptionService) RefreshConfig(ctx context.Context, name, target string, fetchFunc func(string) (string, int, error)) (string, int, error) {
	return s.fetchConfig(ctx, name, fetchFunc)
}

// CreateSubscription 创建订阅
func (s *SubscriptionService) CreateSubscription(ctx context.Context, sub *database.Subscription) error {
	sub.Name = strings.TrimSpace(sub.Name)
	sub.Description = strings.TrimSpace(sub.Description)
	if sub.URL != nil {
		trimmedURL := strings.TrimSpace(*sub.URL)
		sub.URL = &trimmedURL
	}

	// 检查名称是否已存在
	exists, err := s.repo.Exists(ctx, sub.Name)
	if err != nil {
		return fmt.Errorf("检查订阅名称失败: %w", err)
	}
	if exists {
		return fmt.Errorf("订阅名称已存在: %s", sub.Name)
	}

	// 验证 URL
	if sub.URL == nil || strings.TrimSpace(*sub.URL) == "" {
		return fmt.Errorf("订阅必须提供 URL")
	}

	// 设置默认值
	if sub.RefreshInterval == 0 {
		sub.RefreshInterval = 3600 // 默认 1 小时
	}

	return s.repo.Create(ctx, sub)
}

// GetSubscription 获取订阅信息
func (s *SubscriptionService) GetSubscription(ctx context.Context, name string) (*database.Subscription, error) {
	return s.repo.GetByName(ctx, name)
}

// GetSubscriptionByID 获取订阅信息
func (s *SubscriptionService) GetSubscriptionByID(ctx context.Context, id int) (*database.Subscription, error) {
	return s.repo.GetByID(ctx, id)
}

// ListSubscriptions 列出所有订阅（附带流量信息）
func (s *SubscriptionService) ListSubscriptions(ctx context.Context) ([]database.Subscription, error) {
	subs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	// 批量从 Redis 读取流量信息，失败不影响主流程
	for i := range subs {
		if info, err := s.cache.GetUserInfo(ctx, subs[i].Name); err == nil {
			subs[i].UserInfo = info
		}
	}
	return subs, nil
}

// UpdateSubscription 更新订阅
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, sub *database.Subscription) error {
	sub.Name = strings.TrimSpace(sub.Name)
	sub.Description = strings.TrimSpace(sub.Description)
	if sub.URL != nil {
		trimmedURL := strings.TrimSpace(*sub.URL)
		sub.URL = &trimmedURL
	}

	// 验证 URL
	if sub.URL == nil || strings.TrimSpace(*sub.URL) == "" {
		return fmt.Errorf("订阅必须提供 URL")
	}

	// 查旧记录，获取原名称用于清缓存和判断是否改名
	old, err := s.repo.GetByID(ctx, sub.ID)
	if err != nil {
		return err
	}

	// 改名时检查新名称是否冲突
	if old.Name != sub.Name {
		exists, err := s.repo.Exists(ctx, sub.Name)
		if err != nil {
			return fmt.Errorf("检查订阅名称失败: %w", err)
		}
		if exists {
			return fmt.Errorf("订阅名称已存在: %s", sub.Name)
		}
	}

	// 清除旧名称的缓存（含关联的策略配置缓存）
	_ = s.cache.DeleteAll(ctx, old.Name)
	_ = s.cache.DeleteAllByPattern(ctx, "ruleflow:policy:config:*")

	return s.repo.Update(ctx, sub)
}

// DeleteSubscriptionByID 删除订阅
func (s *SubscriptionService) DeleteSubscriptionByID(ctx context.Context, id int) error {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	_ = s.cache.DeleteAll(ctx, sub.Name)
	_ = s.cache.DeleteUserInfo(ctx, sub.Name)
	return s.repo.DeleteByID(ctx, id)
}

// ClearCache 清除缓存
func (s *SubscriptionService) ClearCache(ctx context.Context, name string) error {
	return s.cache.DeleteAll(ctx, name)
}

// ClearAllCache 清除所有缓存
func (s *SubscriptionService) ClearAllCache(ctx context.Context) error {
	return s.cache.DeleteAllByPattern(ctx, "ruleflow:*")
}

// Health 健康检查
func (s *SubscriptionService) Health(ctx context.Context) map[string]interface{} {
	status := make(map[string]interface{})

	// 数据库健康检查
	dbHealth := make(chan map[string]string, 1)
	go func() {
		dbHealth <- s.repo.GetDB().Health()
	}()

	// Redis 健康检查
	cacheHealth := make(chan map[string]string, 1)
	go func() {
		cacheHealth <- s.cache.GetClient().Health()
	}()

	// 收集结果
	select {
	case h := <-dbHealth:
		status["database"] = h
	case <-time.After(2 * time.Second):
		status["database"] = map[string]string{"status": "timeout"}
	}

	select {
	case h := <-cacheHealth:
		status["redis"] = h
	case <-time.After(2 * time.Second):
		status["redis"] = map[string]string{"status": "timeout"}
	}

	// 整体状态
	allHealthy := true
	for _, component := range []string{"database", "redis"} {
		if compStatus, ok := status[component].(map[string]string); ok {
			if compStatus["status"] != "healthy" {
				allHealthy = false
				break
			}
		} else {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		status["status"] = "healthy"
	} else {
		status["status"] = "degraded"
	}

	return status
}
