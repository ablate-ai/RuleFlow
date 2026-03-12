package services

import (
	"context"
	"fmt"

	"github.com/c.chen/ruleflow/database"
)

// TemplateService 模板服务
type TemplateService struct {
	templateRepo *database.TemplateRepo
}

// NewTemplateService 创建模板服务
func NewTemplateService(templateRepo *database.TemplateRepo) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
	}
}

// CreateTemplate 创建模板
func (s *TemplateService) CreateTemplate(ctx context.Context, tpl *database.Template) error {
	// 检查名称是否已存在
	exists, err := s.templateRepo.Exists(ctx, tpl.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("模板名称已存在: %s", tpl.Name)
	}
	return s.templateRepo.Create(ctx, tpl)
}

// GetTemplateByName 根据名称获取模板
func (s *TemplateService) GetTemplateByName(ctx context.Context, name string) (*database.Template, error) {
	return s.templateRepo.GetByName(ctx, name)
}

// GetTemplateByID 根据 ID 获取模板
func (s *TemplateService) GetTemplateByID(ctx context.Context, id int64) (*database.Template, error) {
	return s.templateRepo.GetByID(ctx, id)
}

// ListTemplates 列出所有模板
func (s *TemplateService) ListTemplates(ctx context.Context) ([]database.Template, error) {
	return s.templateRepo.List(ctx)
}

// UpdateTemplate 更新模板
func (s *TemplateService) UpdateTemplate(ctx context.Context, id int64, tpl *database.Template) error {
	current, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if current.Name != tpl.Name {
		exists, err := s.templateRepo.Exists(ctx, tpl.Name)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("模板名称已存在: %s", tpl.Name)
		}
	}
	tpl.ID = id
	return s.templateRepo.Update(ctx, id, tpl)
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(ctx context.Context, id int64) error {
	return s.templateRepo.Delete(ctx, id)
}

// Health 检查服务健康状态
func (s *TemplateService) Health(ctx context.Context) map[string]string {
	status := map[string]string{
		"templates": "available",
	}

	// 简单测试数据库连接
	_, err := s.templateRepo.List(ctx)
	if err != nil {
		status["status"] = "unhealthy"
		status["error"] = err.Error()
	} else {
		status["status"] = "healthy"
	}

	return status
}
