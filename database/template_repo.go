package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Template 模板模型
type Template struct {
	ID          int
	Name        string
	Description string
	Content     string
	IsDefault   bool
	Target      string
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TemplateRepo 模板仓储
type TemplateRepo struct {
	db *DB
}

// NewTemplateRepo 创建模板仓储
func NewTemplateRepo(db *DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

// Create 创建模板
func (r *TemplateRepo) Create(ctx context.Context, tpl *Template) error {
	query := `
		INSERT INTO templates (name, description, content, is_default, target, tags)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		tpl.Name, tpl.Description, tpl.Content, tpl.IsDefault, tpl.Target, tpl.Tags,
	).Scan(&tpl.ID, &tpl.CreatedAt, &tpl.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建模板失败: %w", err)
	}

	// 如果设置为默认模板，需要将其他模板的 is_default 设为 false
	if tpl.IsDefault {
		query = `UPDATE templates SET is_default = false WHERE id != $1`
		_, err = r.db.Pool.Exec(ctx, query, tpl.ID)
		if err != nil {
			return fmt.Errorf("更新默认模板失败: %w", err)
		}
	}

	return nil
}

// GetByName 根据名称获取模板
func (r *TemplateRepo) GetByName(ctx context.Context, name string) (*Template, error) {
	query := `
		SELECT id, name, description, content, is_default, target, tags,
		       created_at, updated_at
		FROM templates
		WHERE name = $1
	`

	tpl := &Template{}
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&tpl.ID,
		&tpl.Name,
		&tpl.Description,
		&tpl.Content,
		&tpl.IsDefault,
		&tpl.Target,
		&tpl.Tags,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("模板不存在: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	return tpl, nil
}

// GetDefault 获取默认模板
func (r *TemplateRepo) GetDefault(ctx context.Context) (*Template, error) {
	query := `
		SELECT id, name, description, content, is_default, tags,
		       created_at, updated_at
		FROM templates
		WHERE is_default = true
	`

	tpl := &Template{}
	err := r.db.Pool.QueryRow(ctx, query).Scan(
		&tpl.ID,
		&tpl.Name,
		&tpl.Description,
		&tpl.Content,
		&tpl.IsDefault,
		&tpl.Tags,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("未找到默认模板")
	}
	if err != nil {
		return nil, fmt.Errorf("查询默认模板失败: %w", err)
	}

	return tpl, nil
}

// List 列出所有模板
func (r *TemplateRepo) List(ctx context.Context) ([]Template, error) {
	query := `
		SELECT id, name, description, content, is_default, target, tags,
		       created_at, updated_at
		FROM templates
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}
	defer rows.Close()

	tpls := []Template{}
	for rows.Next() {
		tpl := Template{}
		err := rows.Scan(
			&tpl.ID,
			&tpl.Name,
			&tpl.Description,
			&tpl.Content,
			&tpl.IsDefault,
			&tpl.Target,
			&tpl.Tags,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描模板行失败: %w", err)
		}
		tpls = append(tpls, tpl)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历模板行失败: %w", err)
	}

	return tpls, nil
}

// Update 更新模板
func (r *TemplateRepo) Update(ctx context.Context, tpl *Template) error {
	query := `
		UPDATE templates
		SET description = $2, content = $3, is_default = $4, target = $5, tags = $6
		WHERE name = $1
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		tpl.Name, tpl.Description, tpl.Content, tpl.IsDefault, tpl.Target, tpl.Tags,
	).Scan(&tpl.UpdatedAt)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("模板不存在: %s", tpl.Name)
	}
	if err != nil {
		return fmt.Errorf("更新模板失败: %w", err)
	}

	// 如果设置为默认模板，需要将其他模板的 is_default 设为 false
	if tpl.IsDefault {
		query = `UPDATE templates SET is_default = false WHERE id != $1`
		_, err = r.db.Pool.Exec(ctx, query, tpl.ID)
		if err != nil {
			return fmt.Errorf("更新默认模板失败: %w", err)
		}
	}

	return nil
}

// Delete 删除模板
func (r *TemplateRepo) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM templates WHERE name = $1`

	result, err := r.db.Pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("模板不存在: %s", name)
	}

	return nil
}

// Exists 检查模板是否存在
func (r *TemplateRepo) Exists(ctx context.Context, name string) (bool, error) {
	query := `SELECT 1 FROM templates WHERE name = $1`

	var exists int
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(&exists)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("检查模板存在性失败: %w", err)
	}

	return true, nil
}

// GetDB 获取数据库实例
func (r *TemplateRepo) GetDB() *DB {
	return r.db
}
