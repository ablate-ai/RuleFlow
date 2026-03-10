package services

import (
	"context"
	"log"
	"time"

	"github.com/c.chen/ruleflow/database"
)

// SubscriptionScheduler 订阅自动刷新调度器
type SubscriptionScheduler struct {
	subRepo     *database.SubscriptionRepo
	syncService *SubscriptionSyncService
	// checkInterval 轮询间隔，默认 1 分钟
	checkInterval time.Duration
}

// NewSubscriptionScheduler 创建调度器
func NewSubscriptionScheduler(subRepo *database.SubscriptionRepo, syncService *SubscriptionSyncService) *SubscriptionScheduler {
	return &SubscriptionScheduler{
		subRepo:       subRepo,
		syncService:   syncService,
		checkInterval: time.Minute,
	}
}

// Start 在后台启动调度循环，ctx 取消时退出
func (s *SubscriptionScheduler) Start(ctx context.Context) {
	go s.loop(ctx)
	log.Println("[scheduler] 订阅自动刷新调度器已启动")
}

func (s *SubscriptionScheduler) loop(ctx context.Context) {
	// 启动时立即执行一次，避免等待第一个 tick
	s.run(ctx)

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[scheduler] 调度器已停止")
			return
		case <-ticker.C:
			s.run(ctx)
		}
	}
}

// run 检查所有需要刷新的订阅并触发同步
func (s *SubscriptionScheduler) run(ctx context.Context) {
	subs, err := s.subRepo.List(ctx)
	if err != nil {
		log.Printf("[scheduler] 获取订阅列表失败: %v", err)
		return
	}

	now := time.Now()
	for _, sub := range subs {
		if !sub.AutoRefresh || !sub.Enabled {
			continue
		}
		interval := time.Duration(sub.RefreshInterval) * time.Second
		if interval <= 0 {
			interval = time.Hour // 兜底默认 1 小时
		}
		// 从未同步过，或距上次同步已超过 interval
		if sub.LastFetchedAt == nil || now.Sub(*sub.LastFetchedAt) >= interval {
			log.Printf("[scheduler] 触发自动同步: %s（间隔 %v）", sub.Name, interval)
			go func(id int, name string) {
				syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				if _, err := s.syncService.SyncSubscription(syncCtx, id); err != nil {
					log.Printf("[scheduler] 自动同步失败 %s: %v", name, err)
				}
			}(sub.ID, sub.Name)
		}
	}
}
