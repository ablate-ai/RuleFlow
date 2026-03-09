package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/c.chen/ruleflow/database"
	"github.com/c.chen/ruleflow/internal/app"
	"github.com/c.chen/ruleflow/services"
)

// policyConfigCache 策略配置缓存接口
type policyConfigCache interface {
	GetPolicyConfig(ctx context.Context, token string) (string, error)
	SetPolicyConfig(ctx context.Context, token, yaml string) error
	DeletePolicyConfig(ctx context.Context, token string) error
}

// Handlers API 处理器
type Handlers struct {
	subscriptionService     *services.SubscriptionService
	templateService         *services.TemplateService
	configPolicyService     *services.ConfigPolicyService
	nodeService             *services.NodeService
	subscriptionSyncService *services.SubscriptionSyncService
	policyCache             policyConfigCache
}

// NewHandlers 创建 API 处理器
func NewHandlers(
	subscriptionService *services.SubscriptionService,
	templateService *services.TemplateService,
	configPolicyService *services.ConfigPolicyService,
	nodeService *services.NodeService,
	subscriptionSyncService *services.SubscriptionSyncService,
	policyCache policyConfigCache,
) *Handlers {
	return &Handlers{
		subscriptionService:     subscriptionService,
		templateService:         templateService,
		configPolicyService:     configPolicyService,
		nodeService:             nodeService,
		subscriptionSyncService: subscriptionSyncService,
		policyCache:             policyCache,
	}
}

// CreateSubscription 创建订阅
func (h *Handlers) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	var sub database.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	ctx := r.Context()
	if err := h.subscriptionService.CreateSubscription(ctx, &sub); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, sub)
}

// GetSubscription 获取订阅信息
func (h *Handlers) GetSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取订阅名称
	// 假设路径格式为 /api/subscriptions/{name}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	ctx := r.Context()
	sub, err := h.subscriptionService.GetSubscription(ctx, name)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, sub)
}

// ListSubscriptions 列出所有订阅
func (h *Handlers) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	ctx := r.Context()
	subs, err := h.subscriptionService.ListSubscriptions(ctx)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, subs)
}

// UpdateSubscription 更新订阅
func (h *Handlers) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取订阅 ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的订阅 ID")
		return
	}

	var sub database.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	sub.ID = id

	ctx := r.Context()
	if err := h.subscriptionService.UpdateSubscription(ctx, &sub); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, sub)
}

// DeleteSubscription 删除订阅
func (h *Handlers) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取订阅名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	ctx := r.Context()
	if err := h.subscriptionService.DeleteSubscription(ctx, name); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, map[string]string{"message": "订阅已删除"})
}

// RefreshSubscription 手动刷新订阅
func (h *Handlers) RefreshSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取订阅名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	_ = parts[4] // 订阅名称

	// 获取目标类型
	_ = r.URL.Query().Get("target")
	// 注意：这里需要传入实际的 fetch 函数，从主程序传入
	// 这是一个简化版本，实际使用中需要从依赖注入获取
	SendError(w, http.StatusNotImplemented, "功能需要集成到主路由")
}

// ClearCache 清除订阅缓存
func (h *Handlers) ClearCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 检查是否清除所有缓存
	if r.URL.Path == "/api/cache" {
		ctx := r.Context()
		if err := h.subscriptionService.ClearAllCache(ctx); err != nil {
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		SendSuccess(w, map[string]string{"message": "所有缓存已清除"})
		return
	}

	// 清除特定订阅的缓存
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	ctx := r.Context()
	if err := h.subscriptionService.ClearCache(ctx, name); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, map[string]string{"message": "缓存已清除"})
}

// Health 健康检查
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	ctx := r.Context()
	health := h.subscriptionService.Health(ctx)

	status := http.StatusOK
	if health["status"] != "healthy" {
		status = http.StatusServiceUnavailable
	}

	SendJSON(w, status, health)
}

// ==================== 模板 API ====================

// CreateTemplate 创建模板
func (h *Handlers) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	var tpl database.Template
	if err := json.NewDecoder(r.Body).Decode(&tpl); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	ctx := r.Context()
	if err := h.templateService.CreateTemplate(ctx, &tpl); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, tpl)
}

// GetTemplate 获取模板信息
func (h *Handlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取模板名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	ctx := r.Context()
	tpl, err := h.templateService.GetTemplateByName(ctx, name)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, tpl)
}

// ListTemplates 列出所有模板
func (h *Handlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	ctx := r.Context()
	tpls, err := h.templateService.ListTemplates(ctx)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, tpls)
}

// UpdateTemplate 更新模板
func (h *Handlers) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取模板名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	var tpl database.Template
	if err := json.NewDecoder(r.Body).Decode(&tpl); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	// 确保名称匹配
	tpl.Name = name

	ctx := r.Context()
	if err := h.templateService.UpdateTemplate(ctx, &tpl); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, tpl)
}

// DeleteTemplate 删除模板
func (h *Handlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取模板名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	ctx := r.Context()
	if err := h.templateService.DeleteTemplate(ctx, name); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, map[string]string{"message": "模板已删除"})
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	YAML      string `json:"yaml"`
	NodeCount int    `json:"node_count"`
	FromCache bool   `json:"from_cache"`
}

// ==================== 配置策略 API ====================

// CreateConfigPolicy 创建配置策略
func (h *Handlers) CreateConfigPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	var policy database.ConfigPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	ctx := r.Context()
	if err := h.configPolicyService.Create(ctx, &policy); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, policy)
}

// GetConfigPolicy 获取配置策略
func (h *Handlers) GetConfigPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取配置策略名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	ctx := r.Context()
	policy, err := h.configPolicyService.GetByName(ctx, name)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, policy)
}

// ListConfigPolicies 列出所有配置策略
func (h *Handlers) ListConfigPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	ctx := r.Context()
	policies, err := h.configPolicyService.List(ctx)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, policies)
}

// UpdateConfigPolicy 更新配置策略
func (h *Handlers) UpdateConfigPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取配置策略名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	var policy database.ConfigPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	// 确保名称匹配
	policy.Name = name

	ctx := r.Context()
	if err := h.configPolicyService.Update(ctx, &policy); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, policy)
}

// DeleteConfigPolicy 删除配置策略
func (h *Handlers) DeleteConfigPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取配置策略名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	name := parts[3]

	ctx := r.Context()
	if err := h.configPolicyService.Delete(ctx, name); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, map[string]string{"message": "配置策略已删除"})
}

// ==================== 节点管理 API ====================

// CreateNode 创建节点（手动添加）
func (h *Handlers) CreateNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	var node database.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	ctx := r.Context()
	if err := h.nodeService.ValidateNode(&node); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.nodeService.AddManualNode(ctx, &node); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, node)
}

// GetNode 获取节点详情
func (h *Handlers) GetNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取节点 ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	var id int
	if _, err := fmt.Sscanf(parts[3], "%d", &id); err != nil {
		SendError(w, http.StatusBadRequest, "无效的节点 ID")
		return
	}

	ctx := r.Context()
	node, err := h.nodeService.GetNode(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, node)
}

// ListNodes 列出节点
func (h *Handlers) ListNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	ctx := r.Context()

	// 构建筛选条件
	filter := database.NodeFilter{}

	// 按来源筛选
	if source := r.URL.Query().Get("source"); source != "" {
		filter.Source = source
	}

	// 按协议筛选
	if protocol := r.URL.Query().Get("protocol"); protocol != "" {
		filter.Protocol = protocol
	}

	// 按启用状态筛选
	if enabled := r.URL.Query().Get("enabled"); enabled != "" {
		enabledBool := enabled == "true" || enabled == "1"
		filter.Enabled = &enabledBool
	}

	// 按标签筛选
	if tags := r.URL.Query().Get("tags"); tags != "" {
		filter.Tags = strings.Split(tags, ",")
	}

	nodes, err := h.nodeService.ListNodes(ctx, filter)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, nodes)
}

// UpdateNode 更新节点
func (h *Handlers) UpdateNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取节点 ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	var id int
	if _, err := fmt.Sscanf(parts[3], "%d", &id); err != nil {
		SendError(w, http.StatusBadRequest, "无效的节点 ID")
		return
	}

	var node database.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	ctx := r.Context()
	if err := h.nodeService.ValidateNode(&node); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.nodeService.UpdateNode(ctx, id, &node); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 获取更新后的节点
	updatedNode, err := h.nodeService.GetNode(ctx, id)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, updatedNode)
}

// DeleteNode 删除节点
func (h *Handlers) DeleteNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取节点 ID
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	var id int
	if _, err := fmt.Sscanf(parts[3], "%d", &id); err != nil {
		SendError(w, http.StatusBadRequest, "无效的节点 ID")
		return
	}

	ctx := r.Context()
	if err := h.nodeService.DeleteNode(ctx, id); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, map[string]string{"message": "节点已删除"})
}

// BatchNodeOperation 批量节点操作
func (h *Handlers) BatchNodeOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	var req struct {
		IDs     []int  `json:"ids"`
		Enabled *bool  `json:"enabled"`
		Action  string `json:"action"` // enable, disable
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	if len(req.IDs) == 0 {
		SendError(w, http.StatusBadRequest, "节点 ID 列表不能为空")
		return
	}

	ctx := r.Context()

	// 根据操作类型执行
	switch req.Action {
	case "enable":
		enabled := true
		count, err := h.nodeService.BatchEnable(ctx, req.IDs, enabled)
		if err != nil {
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		SendSuccess(w, map[string]interface{}{"message": "节点已启用", "count": count})

	case "disable":
		enabled := false
		count, err := h.nodeService.BatchEnable(ctx, req.IDs, enabled)
		if err != nil {
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		SendSuccess(w, map[string]interface{}{"message": "节点已禁用", "count": count})

	default:
		SendError(w, http.StatusBadRequest, "不支持的操作: "+req.Action)
	}
}

// ==================== 订阅同步 API ====================

// SyncSubscription 同步订阅节点
func (h *Handlers) SyncSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取订阅名称
	// 路径格式：/api/subscriptions/{name}/sync
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	subscriptionName := parts[3]

	ctx := r.Context()
	count, err := h.subscriptionSyncService.SyncSubscription(ctx, subscriptionName)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, map[string]interface{}{
		"message":    "订阅同步成功",
		"node_count": count,
	})
}

// GetSubscriptionSyncStatus 获取订阅同步状态
func (h *Handlers) GetSubscriptionSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	// 从 URL 路径中提取订阅名称
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		SendError(w, http.StatusBadRequest, "无效的路径")
		return
	}
	subscriptionName := parts[3]

	ctx := r.Context()
	status, err := h.subscriptionSyncService.GetSyncStatus(ctx, subscriptionName)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, status)
}

// GetNodeStats 获取节点统计信息
func (h *Handlers) GetNodeStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "方法不允许")
		return
	}

	ctx := r.Context()
	stats, err := h.nodeService.GetNodeStats(ctx)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, stats)
}

// GenerateConfig 根据配置策略 token 生成 YAML 配置（带 Redis 缓存）
// 路由: GET /config?token=xxx
func (h *Handlers) GenerateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "缺少 token 参数", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// 优先从 Redis 缓存读取
	if h.policyCache != nil {
		if cached, err := h.policyCache.GetPolicyConfig(ctx, token); err == nil && cached != "" {
			w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
			w.Header().Set("X-Cache", "HIT")
			fmt.Fprint(w, cached)
			return
		}
	}

	// 根据 token 查找策略
	policy, err := h.configPolicyService.GetByToken(ctx, token)
	if err != nil {
		http.Error(w, "无效的 token", http.StatusNotFound)
		return
	}

	if !policy.Enabled {
		http.Error(w, "该配置策略已禁用", http.StatusForbidden)
		return
	}

	// 获取节点
	dbNodes, err := h.configPolicyService.GetNodesForPolicy(ctx, policy)
	if err != nil {
		http.Error(w, "获取节点失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if len(dbNodes) == 0 {
		http.Error(w, "该配置策略下没有可用节点，请先同步订阅源", http.StatusServiceUnavailable)
		return
	}

	// 获取模板内容
	var templateContent string
	if policy.TemplateName != "" {
		if tpl, err := h.templateService.GetTemplateByName(ctx, policy.TemplateName); err == nil {
			templateContent = tpl.Content
		}
	}

	// 将数据库节点转换为 ProxyNode
	proxyNodes := make([]*app.ProxyNode, 0, len(dbNodes))
	for _, n := range dbNodes {
		proxyNodes = append(proxyNodes, &app.ProxyNode{
			Protocol: n.Protocol,
			Name:     n.Name,
			Server:   n.Server,
			Port:     n.Port,
			Options:  n.Config,
		})
	}

	// 生成 YAML
	target := policy.Target
	if target == "" {
		target = "clash"
	}
	var yamlContent string
	if templateContent != "" {
		yamlContent, err = app.BuildYAMLFromTemplateContent(proxyNodes, templateContent, target)
	} else {
		yamlContent, err = app.BuildYAMLFromDefaultTemplate(proxyNodes, target)
	}
	if err != nil {
		http.Error(w, "生成配置失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 写入 Redis 缓存
	if h.policyCache != nil {
		_ = h.policyCache.SetPolicyConfig(ctx, token, yamlContent)
	}

	filename := "clash_config.yaml"
	if target == "stash" {
		filename = "stash_config.yaml"
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("X-Node-Count", fmt.Sprintf("%d", len(proxyNodes)))
	w.Header().Set("X-Cache", "MISS")
	fmt.Fprint(w, yamlContent)
}
