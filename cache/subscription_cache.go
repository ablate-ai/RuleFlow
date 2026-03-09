package cache

import (
	"context"
	"fmt"
	"time"
)

// SubscriptionCache 订阅缓存管理
type SubscriptionCache struct {
	client *Client
	ttl    time.Duration
}

// NewSubscriptionCache 创建订阅缓存
func NewSubscriptionCache(client *Client, ttl time.Duration) *SubscriptionCache {
	return &SubscriptionCache{
		client: client,
		ttl:    ttl,
	}
}

// 键名格式常量
const (
	// CacheKeySubContent 订阅内容缓存键格式：ruleflow:sub:content:<name>
	CacheKeySubContent = "ruleflow:sub:content:%s"
	// CacheKeyConfig 配置缓存键格式：ruleflow:config:<name>:<target>
	CacheKeyConfig = "ruleflow:config:%s:%s"
	// CacheKeyLock 分布式锁键格式：ruleflow:lock:fetch:<name>
	CacheKeyLock = "ruleflow:lock:fetch:%s"
	// CacheKeyPolicyConfig 策略配置缓存键格式：ruleflow:policy:config:<token>
	CacheKeyPolicyConfig = "ruleflow:policy:config:%s"
)

// CachedConfig 缓存的配置数据
type CachedConfig struct {
	YAML      string    `json:"yaml"`
	NodeCount int       `json:"node_count"`
	FetchedAt time.Time `json:"fetched_at"`
}

// GetConfig 获取缓存的配置
func (c *SubscriptionCache) GetConfig(ctx context.Context, name, target string) (*CachedConfig, error) {
	key := fmt.Sprintf(CacheKeyConfig, name, target)
	data, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// 这里简化处理，实际使用中应该解析 JSON
	// 为了保持简单，我们直接存储 YAML 字符串，使用分隔符分隔元数据
	return &CachedConfig{
		YAML: data,
	}, nil
}

// SetConfig 设置配置缓存
func (c *SubscriptionCache) SetConfig(ctx context.Context, name, target string, yaml string, nodeCount int) error {
	key := fmt.Sprintf(CacheKeyConfig, name, target)
	return c.client.Set(ctx, key, yaml, c.ttl)
}

// GetContent 获取缓存的订阅内容
func (c *SubscriptionCache) GetContent(ctx context.Context, name string) (string, error) {
	key := fmt.Sprintf(CacheKeySubContent, name)
	return c.client.Get(ctx, key)
}

// SetContent 设置订阅内容缓存
func (c *SubscriptionCache) SetContent(ctx context.Context, name, content string) error {
	key := fmt.Sprintf(CacheKeySubContent, name)
	return c.client.Set(ctx, key, content, c.ttl)
}

// DeleteConfig 删除配置缓存
func (c *SubscriptionCache) DeleteConfig(ctx context.Context, name, target string) error {
	key := fmt.Sprintf(CacheKeyConfig, name, target)
	return c.client.Delete(ctx, key)
}

// DeleteAll 删除所有相关缓存
func (c *SubscriptionCache) DeleteAll(ctx context.Context, name string) error {
	// 删除内容缓存
	contentKey := fmt.Sprintf(CacheKeySubContent, name)

	// 删除所有目标的配置缓存
	clashKey := fmt.Sprintf(CacheKeyConfig, name, "clash")
	stashKey := fmt.Sprintf(CacheKeyConfig, name, "stash")

	return c.client.Delete(ctx, contentKey, clashKey, stashKey)
}

// DeleteAllByPattern 按模式删除所有缓存
func (c *SubscriptionCache) DeleteAllByPattern(ctx context.Context, pattern string) error {
	// 使用 Keys 命令查找匹配的键
	keys, err := c.client.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("查找键失败: %w", err)
	}

	if len(keys) > 0 {
		return c.client.Delete(ctx, keys...)
	}

	return nil
}

// GetPolicyConfig 获取策略配置缓存（按 token）
func (c *SubscriptionCache) GetPolicyConfig(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf(CacheKeyPolicyConfig, token)
	return c.client.Get(ctx, key)
}

// SetPolicyConfig 设置策略配置缓存
func (c *SubscriptionCache) SetPolicyConfig(ctx context.Context, token, yaml string) error {
	key := fmt.Sprintf(CacheKeyPolicyConfig, token)
	return c.client.Set(ctx, key, yaml, c.ttl)
}

// DeletePolicyConfig 删除策略配置缓存
func (c *SubscriptionCache) DeletePolicyConfig(ctx context.Context, token string) error {
	key := fmt.Sprintf(CacheKeyPolicyConfig, token)
	return c.client.Delete(ctx, key)
}

// AcquireFetchLock 获取订阅获取锁（防止并发刷新）
func (c *SubscriptionCache) AcquireFetchLock(ctx context.Context, name string) (bool, error) {
	key := fmt.Sprintf(CacheKeyLock, name)
	return c.client.AcquireLock(ctx, key, 30*time.Second)
}

// ReleaseFetchLock 释放订阅获取锁
func (c *SubscriptionCache) ReleaseFetchLock(ctx context.Context, name string) error {
	key := fmt.Sprintf(CacheKeyLock, name)
	return c.client.ReleaseLock(ctx, key)
}

// FlushAll 清空所有缓存（谨慎使用）
func (c *SubscriptionCache) FlushAll(ctx context.Context) error {
	return c.client.Client.FlushDB(ctx).Err()
}

// GetClient 获取 Redis 客户端
func (c *SubscriptionCache) GetClient() *Client {
	return c.client
}
