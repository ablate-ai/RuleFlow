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

// GetConfig 获取订阅配置（带缓存）
func (s *SubscriptionService) GetConfig(ctx context.Context, name, target string, fetchFunc func(string) (string, int, error)) (string, int, error) {
	// 1. 尝试从缓存获取
	cached, err := s.cache.GetConfig(ctx, name, target)
	if err == nil && cached.YAML != "" {
		return cached.YAML, 0, nil // 从缓存返回
	}

	// 2. 检查是否可以获取锁（防止并发刷新）
	acquired, err := s.cache.AcquireFetchLock(ctx, name)
	if err != nil {
		// Redis 错误，降级为直接获取
		return s.fetchAndCacheConfig(ctx, name, target, fetchFunc)
	}

	if !acquired {
		// 其他实例正在获取，等待一小段时间后重试缓存
		select {
		case <-time.After(500 * time.Millisecond):
			cached, err := s.cache.GetConfig(ctx, name, target)
			if err == nil && cached.YAML != "" {
				return cached.YAML, 0, nil
			}
			return "", 0, fmt.Errorf("订阅正在刷新中，请稍后再试")
		case <-ctx.Done():
			return "", 0, ctx.Err()
		}
	}

	// 3. 获取锁成功，执行获取逻辑
	defer s.cache.ReleaseFetchLock(ctx, name)

	return s.fetchAndCacheConfig(ctx, name, target, fetchFunc)
}

// fetchAndCacheConfig 获取并缓存配置
func (s *SubscriptionService) fetchAndCacheConfig(ctx context.Context, name, target string, fetchFunc func(string) (string, int, error)) (string, int, error) {
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

	// 缓存配置
	_ = s.cache.SetConfig(ctx, name, target, yaml, nodeCount)

	return yaml, nodeCount, nil
}

// RefreshConfig 手动刷新订阅配置
func (s *SubscriptionService) RefreshConfig(ctx context.Context, name, target string, fetchFunc func(string) (string, int, error)) (string, int, error) {
	// 清除缓存
	_ = s.cache.DeleteConfig(ctx, name, target)

	// 重新获取配置
	return s.GetConfig(ctx, name, target, fetchFunc)
}

// CreateSubscription 创建订阅
func (s *SubscriptionService) CreateSubscription(ctx context.Context, sub *database.Subscription) error {
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
	if sub.Target == "" {
		sub.Target = "clash"
	}
	if sub.RefreshInterval == 0 {
		sub.RefreshInterval = 3600 // 默认 1 小时
	}

	return s.repo.Create(ctx, sub)
}

// GetSubscription 获取订阅信息
func (s *SubscriptionService) GetSubscription(ctx context.Context, name string) (*database.Subscription, error) {
	return s.repo.GetByName(ctx, name)
}

// ListSubscriptions 列出所有订阅
func (s *SubscriptionService) ListSubscriptions(ctx context.Context) ([]database.Subscription, error) {
	return s.repo.List(ctx)
}

// UpdateSubscription 更新订阅
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, sub *database.Subscription) error {
	// 验证 URL
	if sub.URL == nil || strings.TrimSpace(*sub.URL) == "" {
		return fmt.Errorf("订阅必须提供 URL")
	}

	// 清除相关缓存
	_ = s.cache.DeleteAll(ctx, sub.Name)

	return s.repo.Update(ctx, sub)
}

// DeleteSubscription 删除订阅
func (s *SubscriptionService) DeleteSubscription(ctx context.Context, name string) error {
	// 清除相关缓存
	_ = s.cache.DeleteAll(ctx, name)

	return s.repo.Delete(ctx, name)
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
