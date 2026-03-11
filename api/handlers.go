package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/c.chen/ruleflow/database"
	"github.com/c.chen/ruleflow/internal/app"
	"github.com/c.chen/ruleflow/services"
)

// urlParamInt 从 chi 路由参数中提取整数 ID
func urlParamInt(r *http.Request, key string) (int, error) {
	return strconv.Atoi(chi.URLParam(r, key))
}

// policyConfigCache 策略配置缓存接口
type policyConfigCache interface {
	GetPolicyConfig(ctx context.Context, token string) (string, error)
	SetPolicyConfig(ctx context.Context, token, yaml string) error
	DeletePolicyConfig(ctx context.Context, token string) error
	DeleteAllByPattern(ctx context.Context, pattern string) error
}

// Handlers API 处理器
type Handlers struct {
	subscriptionService     *services.SubscriptionService
	templateService         *services.TemplateService
	configPolicyService     *services.ConfigPolicyService
	ruleSourceService       *services.RuleSourceService
	nodeService             *services.NodeService
	subscriptionSyncService *services.SubscriptionSyncService
	ruleSourceSyncService   *services.RuleSourceSyncService
	policyCache             policyConfigCache
}

// NewHandlers 创建 API 处理器
func NewHandlers(
	subscriptionService *services.SubscriptionService,
	templateService *services.TemplateService,
	configPolicyService *services.ConfigPolicyService,
	ruleSourceService *services.RuleSourceService,
	nodeService *services.NodeService,
	subscriptionSyncService *services.SubscriptionSyncService,
	ruleSourceSyncService *services.RuleSourceSyncService,
	policyCache policyConfigCache,
) *Handlers {
	return &Handlers{
		subscriptionService:     subscriptionService,
		templateService:         templateService,
		configPolicyService:     configPolicyService,
		ruleSourceService:       ruleSourceService,
		nodeService:             nodeService,
		subscriptionSyncService: subscriptionSyncService,
		ruleSourceSyncService:   ruleSourceSyncService,
		policyCache:             policyCache,
	}
}

// CreateSubscription 创建订阅
func (h *Handlers) CreateSubscription(w http.ResponseWriter, r *http.Request) {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的订阅 ID")
		return
	}

	ctx := r.Context()
	sub, err := h.subscriptionService.GetSubscriptionByID(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, sub)
}

// ListSubscriptions 列出所有订阅
func (h *Handlers) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
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
	id, err := urlParamInt(r, "id")
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的订阅 ID")
		return
	}

	ctx := r.Context()
	if err := h.subscriptionService.DeleteSubscriptionByID(ctx, id); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, map[string]string{"message": "订阅已删除"})
}

// Health 健康检查
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的模板 ID")
		return
	}

	ctx := r.Context()
	tpl, err := h.templateService.GetTemplateByID(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, tpl)
}

// ListTemplates 列出所有模板
func (h *Handlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的模板 ID")
		return
	}

	var tpl database.Template
	if err := json.NewDecoder(r.Body).Decode(&tpl); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	if strings.TrimSpace(tpl.Name) == "" {
		SendError(w, http.StatusBadRequest, "模板名称不能为空")
		return
	}

	ctx := r.Context()
	if err := h.templateService.UpdateTemplate(ctx, id, &tpl); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 模板更新后清除所有策略配置缓存
	if h.policyCache != nil {
		_ = h.policyCache.DeleteAllByPattern(ctx, "ruleflow:policy:config:*")
	}

	SendSuccess(w, tpl)
}

// DeleteTemplate 删除模板
func (h *Handlers) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的模板 ID")
		return
	}

	ctx := r.Context()
	if err := h.templateService.DeleteTemplate(ctx, id); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	// 模板删除后清除所有策略配置缓存
	if h.policyCache != nil {
		_ = h.policyCache.DeleteAllByPattern(ctx, "ruleflow:policy:config:*")
	}

	SendSuccess(w, map[string]string{"message": "模板已删除"})
}

// ==================== 规则源 API ====================

func (h *Handlers) CreateRuleSource(w http.ResponseWriter, r *http.Request) {
	var source database.RuleSource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	ctx := r.Context()
	if err := h.ruleSourceService.Create(ctx, &source); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, source)
}

func (h *Handlers) GetRuleSource(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的规则源 ID")
		return
	}

	ctx := r.Context()
	source, err := h.ruleSourceService.GetByID(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, source)
}

func (h *Handlers) ListRuleSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sources, err := h.ruleSourceService.List(ctx)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendSuccess(w, sources)
}

func (h *Handlers) UpdateRuleSource(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的规则源 ID")
		return
	}

	var source database.RuleSource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	source.ID = id

	ctx := r.Context()
	if err := h.ruleSourceService.Update(ctx, &source); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, source)
}

func (h *Handlers) DeleteRuleSource(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的规则源 ID")
		return
	}
	ctx := r.Context()
	if err := h.ruleSourceService.Delete(ctx, id); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}
	SendSuccess(w, map[string]string{"message": "规则源已删除"})
}

func (h *Handlers) SyncRuleSource(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的规则源 ID")
		return
	}
	ctx := r.Context()
	count, err := h.ruleSourceSyncService.SyncRuleSource(ctx, id)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendSuccess(w, map[string]interface{}{
		"message":    "规则源同步完成",
		"rule_count": count,
	})
}

func (h *Handlers) ExportRuleSource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if strings.TrimSpace(name) == "" {
		http.Error(w, "缺少规则源名称", http.StatusBadRequest)
		return
	}
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target == "" {
		target = "sing-box"
	}

	ctx := r.Context()
	content, err := h.ruleSourceSyncService.ExportRuleSource(ctx, name, target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch target {
	case "sing-box":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case "clash-classical", "clash-domain", "clash-ipcidr", "surge":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	fmt.Fprint(w, content)
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的策略 ID")
		return
	}

	ctx := r.Context()
	policy, err := h.configPolicyService.GetByID(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, policy)
}

// ListConfigPolicies 列出所有配置策略
func (h *Handlers) ListConfigPolicies(w http.ResponseWriter, r *http.Request) {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的策略 ID")
		return
	}

	var policy database.ConfigPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	policy.ID = id

	ctx := r.Context()
	if err := h.configPolicyService.Update(ctx, &policy); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendSuccess(w, policy)
}

// DeleteConfigPolicy 删除配置策略
func (h *Handlers) DeleteConfigPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的策略 ID")
		return
	}

	ctx := r.Context()
	if err := h.configPolicyService.Delete(ctx, id); err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	SendSuccess(w, map[string]string{"message": "配置策略已删除"})
}

// ClearPolicyConfigCache 清除指定策略的生成配置缓存
func (h *Handlers) ClearPolicyConfigCache(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的策略 ID")
		return
	}

	ctx := r.Context()
	policy, err := h.configPolicyService.GetByID(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, "策略不存在")
		return
	}

	if h.policyCache != nil && policy.Token != "" {
		_ = h.policyCache.DeletePolicyConfig(ctx, policy.Token)
	}

	SendSuccess(w, map[string]string{"message": "配置缓存已清除"})
}

// ==================== 节点管理 API ====================

// ImportNodes 通过 URL 批量导入节点
func (h *Handlers) ImportNodes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	lines := strings.Split(req.Content, "\n")
	type importErr struct {
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	var created int
	var errors []importErr

	ctx := r.Context()
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		proxyNode, err := app.ParseNodeURL(line)
		if err != nil {
			errors = append(errors, importErr{URL: line, Error: err.Error()})
			continue
		}
		node := &database.Node{
			Name:     proxyNode.Name,
			Protocol: proxyNode.Protocol,
			Server:   proxyNode.Server,
			Port:     proxyNode.Port,
			Config:   proxyNode.Options,
		}
		if err := h.nodeService.AddManualNode(ctx, node); err != nil {
			errors = append(errors, importErr{URL: line, Error: err.Error()})
			continue
		}
		created++
	}

	SendSuccess(w, map[string]interface{}{
		"created": created,
		"errors":  errors,
	})
}

// CreateNode 创建节点（手动添加）
func (h *Handlers) CreateNode(w http.ResponseWriter, r *http.Request) {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的订阅 ID")
		return
	}

	ctx := r.Context()
	sub, err := h.subscriptionService.GetSubscriptionByID(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	count, err := h.subscriptionSyncService.SyncSubscription(ctx, sub.ID)
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
	id, err := urlParamInt(r, "id")
	if err != nil {
		SendError(w, http.StatusBadRequest, "无效的订阅 ID")
		return
	}

	ctx := r.Context()
	sub, err := h.subscriptionService.GetSubscriptionByID(ctx, id)
	if err != nil {
		SendError(w, http.StatusNotFound, err.Error())
		return
	}

	status, err := h.subscriptionSyncService.GetSyncStatus(ctx, sub.ID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, status)
}

// GetNodeStats 获取节点统计信息
func (h *Handlers) GetNodeStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := h.nodeService.GetNodeStats(ctx)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendSuccess(w, stats)
}

// GenerateConfig 根据配置策略 token 生成 YAML 配置（带 Redis 缓存）
// 路由: GET /subscribe?token=xxx
func (h *Handlers) GenerateConfig(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "缺少 token 参数", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// 优先从 Redis 缓存读取
	if h.policyCache != nil {
		if cached, err := h.policyCache.GetPolicyConfig(ctx, token); err == nil && cached != "" {
			if policy, err := h.configPolicyService.GetByToken(ctx, token); err == nil {
				switch policy.Target {
				case "surge":
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					cached = finalizeConfigContent(r, "surge", cached)
				case "sing-box":
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
				default:
					w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
				}
			} else {
				w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
			}
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
		dialerProxy := ""
		if v, ok := n.Config["dialer-proxy"].(string); ok {
			dialerProxy = v
		} else if v, ok := n.Config["underlying-proxy"].(string); ok {
			dialerProxy = v
		}
		proxyNodes = append(proxyNodes, &app.ProxyNode{
			Protocol:    n.Protocol,
			Name:        n.Name,
			Server:      n.Server,
			Port:        n.Port,
			Options:     n.Config,
			DialerProxy: dialerProxy,
		})
	}

	// 生成配置
	target := policy.Target
	if target == "" {
		target = "clash-meta"
	}
	var configContent string
	if target == "surge" {
		if templateContent != "" {
			templateContent = replaceTemplateRuntimePlaceholders(r, templateContent)
			configContent, err = app.BuildSurgeFromTemplateContent(proxyNodes, templateContent)
		} else {
			configContent, err = app.BuildSurgeFromDefaultTemplate(proxyNodes)
			configContent = replaceTemplateRuntimePlaceholders(r, configContent)
		}
	} else if target == "sing-box" {
		if templateContent != "" {
			templateContent = replaceTemplateRuntimePlaceholders(r, templateContent)
			configContent, err = app.BuildSingBoxFromTemplateContent(proxyNodes, templateContent)
		} else {
			configContent, err = app.BuildSingBoxFromDefaultTemplate(proxyNodes)
			configContent = replaceTemplateRuntimePlaceholders(r, configContent)
		}
	} else {
		if templateContent != "" {
			templateContent = replaceTemplateRuntimePlaceholders(r, templateContent)
			configContent, err = app.BuildYAMLFromTemplateContent(proxyNodes, templateContent, target)
		} else {
			configContent, err = app.BuildYAMLFromDefaultTemplate(proxyNodes, target)
			configContent = replaceTemplateRuntimePlaceholders(r, configContent)
		}
	}
	if err != nil {
		http.Error(w, "生成配置失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 写入 Redis 缓存
	if h.policyCache != nil {
		_ = h.policyCache.SetPolicyConfig(ctx, token, configContent)
	}

	var filename string
	switch target {
	case "stash":
		filename = "stash_config.yaml"
	case "surge":
		filename = "surge_config.conf"
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case "sing-box":
		filename = "sing_box_config.json"
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case "clash-meta":
		filename = "clash_meta_config.yaml"
	default:
		filename = "clash_meta_config.yaml"
	}
	if target != "surge" && target != "sing-box" {
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("X-Node-Count", fmt.Sprintf("%d", len(proxyNodes)))
	w.Header().Set("X-Cache", "MISS")
	fmt.Fprint(w, finalizeConfigContent(r, target, configContent))
}

func replaceTemplateRuntimePlaceholders(r *http.Request, content string) string {
	if content == "" {
		return content
	}
	baseURL := requestBaseURLString(r)
	if baseURL == "" {
		return content
	}
	ruleSetPathPattern := regexp.MustCompile(`/rulesets/[^\s",]+`)
	return ruleSetPathPattern.ReplaceAllStringFunc(content, func(path string) string {
		return baseURL + path
	})
}
