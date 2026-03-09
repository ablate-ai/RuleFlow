package cache

import (
	"context"
	"encoding/json"
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
	// CacheKeyLock 分布式锁键格式：ruleflow:lock:fetch:<name>
	CacheKeyLock = "ruleflow:lock:fetch:%s"
	// CacheKeyPolicyConfig 策略配置缓存键格式：ruleflow:policy:config:<token>
	CacheKeyPolicyConfig = "ruleflow:policy:config:%s"
	// CacheKeySubUserInfo 订阅流量信息缓存键格式：ruleflow:sub:userinfo:<name>
	CacheKeySubUserInfo = "ruleflow:sub:userinfo:%s"
)

// UserInfo 订阅流量信息（来自响应头 Subscription-Userinfo）
type UserInfo struct {
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
	Total    int64  `json:"total"`
	Expire   *int64 `json:"expire,omitempty"`
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

// GetUserInfo 获取订阅流量信息
func (c *SubscriptionCache) GetUserInfo(ctx context.Context, name string) (*UserInfo, error) {
	key := fmt.Sprintf(CacheKeySubUserInfo, name)
	data, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var info UserInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("解析流量信息失败: %w", err)
	}
	return &info, nil
}

// SetUserInfo 设置订阅流量信息（TTL 与 expire 时间戳对齐，若无则用默认 TTL）
func (c *SubscriptionCache) SetUserInfo(ctx context.Context, name string, info *UserInfo) error {
	key := fmt.Sprintf(CacheKeySubUserInfo, name)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("序列化流量信息失败: %w", err)
	}
	ttl := c.ttl
	if info.Expire != nil {
		remaining := time.Until(time.Unix(*info.Expire, 0))
		if remaining > 0 {
			ttl = remaining
		}
	}
	return c.client.Set(ctx, key, string(data), ttl)
}

// DeleteUserInfo 删除订阅流量信息缓存
func (c *SubscriptionCache) DeleteUserInfo(ctx context.Context, name string) error {
	key := fmt.Sprintf(CacheKeySubUserInfo, name)
	return c.client.Delete(ctx, key)
}

// DeleteAll 删除订阅相关缓存（不含流量信息，流量信息仅在删除订阅时清除）
func (c *SubscriptionCache) DeleteAll(ctx context.Context, name string) error {
	contentKey := fmt.Sprintf(CacheKeySubContent, name)
	return c.client.Delete(ctx, contentKey)
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
