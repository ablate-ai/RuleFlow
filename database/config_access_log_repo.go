package database

import (
	"context"
	"fmt"
	"net/netip"
	"time"
)

// ConfigAccessLog 订阅配置访问日志
type ConfigAccessLog struct {
	ID           int64     `json:"id"`
	PolicyID     *int      `json:"policy_id,omitempty"`
	Token        string    `json:"token,omitempty"`
	ClientIP     *string   `json:"client_ip,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	StatusCode   int       `json:"status_code"`
	Success      bool      `json:"success"`
	CacheHit     bool      `json:"cache_hit"`
	NodeCount    *int      `json:"node_count,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ConfigAccessLogRepo 订阅配置访问日志仓储
type ConfigAccessLogRepo struct {
	db *DB
}

// NewConfigAccessLogRepo 创建访问日志仓储
func NewConfigAccessLogRepo(db *DB) *ConfigAccessLogRepo {
	return &ConfigAccessLogRepo{db: db}
}

// Create 写入访问日志
func (r *ConfigAccessLogRepo) Create(ctx context.Context, log *ConfigAccessLog) error {
	query := `
		INSERT INTO config_access_logs (policy_id, token, client_ip, user_agent, status_code, success, cache_hit, node_count, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''))
		RETURNING id, created_at
	`

	var clientIP any
	if log.ClientIP != nil && *log.ClientIP != "" {
		if parsed, err := netip.ParseAddr(*log.ClientIP); err == nil {
			clientIP = parsed.String()
		}
	}

	err := r.db.Pool.QueryRow(ctx, query,
		log.PolicyID,
		log.Token,
		clientIP,
		log.UserAgent,
		log.StatusCode,
		log.Success,
		log.CacheHit,
		log.NodeCount,
		log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return fmt.Errorf("写入访问日志失败: %w", err)
	}

	return nil
}

// ListByPolicy 按策略查询最近访问日志
func (r *ConfigAccessLogRepo) ListByPolicy(ctx context.Context, policyID int, limit int) ([]*ConfigAccessLog, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT id, policy_id, token, host(client_ip), user_agent, status_code, success, cache_hit, node_count, COALESCE(error_message, ''), created_at
		FROM config_access_logs
		WHERE policy_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("查询访问日志失败: %w", err)
	}
	defer rows.Close()

	logs := make([]*ConfigAccessLog, 0)
	for rows.Next() {
		var log ConfigAccessLog
		var policyIDValue *int
		var clientIP *string
		err := rows.Scan(
			&log.ID,
			&policyIDValue,
			&log.Token,
			&clientIP,
			&log.UserAgent,
			&log.StatusCode,
			&log.Success,
			&log.CacheHit,
			&log.NodeCount,
			&log.ErrorMessage,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描访问日志失败: %w", err)
		}
		log.PolicyID = policyIDValue
		log.ClientIP = clientIP
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历访问日志失败: %w", err)
	}

	return logs, nil
}
