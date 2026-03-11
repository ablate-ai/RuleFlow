package services

import (
	"context"
	"time"
)

type RuleSourceScheduler struct {
	syncService   *RuleSourceSyncService
	checkInterval time.Duration
}

func NewRuleSourceScheduler(syncService *RuleSourceSyncService) *RuleSourceScheduler {
	return &RuleSourceScheduler{
		syncService:   syncService,
		checkInterval: time.Minute,
	}
}

func (s *RuleSourceScheduler) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.syncService.SyncDueSources(ctx)
			}
		}
	}()
}
