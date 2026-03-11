package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/c.chen/ruleflow/api"
	"github.com/c.chen/ruleflow/cache"
	"github.com/c.chen/ruleflow/config"
	"github.com/c.chen/ruleflow/database"
	"github.com/c.chen/ruleflow/internal/app"
	"github.com/c.chen/ruleflow/services"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库和 Redis（数据库必需，Redis 可选）
	var redisClient *cache.Client
	var subscriptionService *services.SubscriptionService

	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		log.Fatal("❌ DATABASE_URL 未设置，RuleFlow 需要 PostgreSQL 才能启动")
	}

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v\n", err)
	}
	log.Printf("✅ 数据库连接成功\n")
	defer db.Close()

	// 尝试连接 Redis
	if os.Getenv("REDIS_ADDR") != "" {
		var err error
		redisClient, err = cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		if err != nil {
			log.Printf("⚠️ Redis 连接失败（将使用无缓存模式）: %v\n", err)
		} else {
			log.Printf("✅ Redis 连接成功\n")
			defer redisClient.Close()
		}
	}

	// 创建订阅服务（数据库可用即可；Redis 仅用于策略配置缓存）
	var templateService *services.TemplateService
	var configPolicyService *services.ConfigPolicyService
	var ruleSourceService *services.RuleSourceService
	var nodeService *services.NodeService
	var subscriptionSyncService *services.SubscriptionSyncService
	var ruleSourceSyncService *services.RuleSourceSyncService
	var subscriptionCache *cache.SubscriptionCache
	if redisClient != nil {
		subscriptionCache = cache.NewSubscriptionCache(redisClient, time.Duration(cfg.CacheTTLSeconds)*time.Second)
	}
	subscriptionRepo := database.NewSubscriptionRepo(db)
	subscriptionService = services.NewSubscriptionService(subscriptionRepo, subscriptionCache)

	// 创建模板服务
	templateRepo := database.NewTemplateRepo(db)
	templateService = services.NewTemplateService(templateRepo)

	// 创建配置策略服务
	configPolicyRepo := database.NewConfigPolicyRepo(db)
	configAccessLogRepo := database.NewConfigAccessLogRepo(db)
	nodeRepo := database.NewNodeRepo(db)
	configPolicyService = services.NewConfigPolicyService(configPolicyRepo, configAccessLogRepo, subscriptionRepo, nodeRepo)

	// 创建规则源服务
	ruleSourceRepo := database.NewRuleSourceRepo(db)
	ruleSourceService = services.NewRuleSourceService(ruleSourceRepo)
	ruleSourceSyncService = services.NewRuleSourceSyncService(ruleSourceRepo)

	// 创建节点服务
	nodeService = services.NewNodeService(nodeRepo)

	// 创建订阅同步服务
	subscriptionSyncService = services.NewSubscriptionSyncService(subscriptionRepo, nodeRepo)

	// 启动自动刷新调度器
	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	if subscriptionSyncService != nil {
		scheduler := services.NewSubscriptionScheduler(database.NewSubscriptionRepo(db), subscriptionSyncService)
		scheduler.Start(schedulerCtx)
	}
	if ruleSourceSyncService != nil {
		ruleSourceScheduler := services.NewRuleSourceScheduler(ruleSourceSyncService)
		ruleSourceScheduler.Start(schedulerCtx)
	}
	// 启动日志清理调度器
	logCleanupScheduler := services.NewLogCleanupScheduler(configAccessLogRepo,
		services.WithLogCleanupKeepDays(cfg.LogKeepDays),
		services.WithLogCleanupMaxRecords(cfg.LogMaxRecords),
		services.WithLogCleanupCheckInterval(time.Duration(cfg.LogCheckInterval)*time.Hour),
	)
	logCleanupScheduler.Start(schedulerCtx)

	// 创建 API 处理器
	apiHandlers := api.NewHandlers(subscriptionService, templateService, configPolicyService, ruleSourceService, nodeService, subscriptionSyncService, ruleSourceSyncService, subscriptionCache)

	// 设置路由
	setupRoutes(cfg, apiHandlers)

	// 启动服务器
	port := cfg.Port
	log.Printf("🚀 服务器启动: http://localhost:%s\n", port)
	log.Printf("💡 管理界面: http://localhost:%s/dashboard\n", port)
	log.Printf("💡 管理接口: http://localhost:%s/api/subscriptions\n", port)
	log.Printf("💡 模板接口: http://localhost:%s/api/templates\n", port)
	log.Printf("💡 规则源接口: http://localhost:%s/api/rule-sources\n", port)
	log.Printf("💡 健康检查: http://localhost:%s/health\n", port)

	// 优雅关闭
	r := setupRoutes(cfg, apiHandlers)
	server := &http.Server{
		Addr:    ":" + port,
		Handler: api.LoggingMiddleware(api.CORSMiddleware(api.RecoveryMiddleware(r))),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ 服务器错误: %v\n", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 收到第二次信号时强制退出
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	log.Println("🛑 正在关闭服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ 优雅关闭超时，强制关闭: %v\n", err)
		_ = server.Close()
	}

	log.Println("✅ 服务器已关闭")
	os.Exit(0)
}

func setupRoutes(cfg *config.Config, apiHandlers *api.Handlers) chi.Router {
	if cfg.AdminPassword != "" {
		log.Printf("🔒 Web 控制台鉴权已启用\n")
	} else {
		log.Printf("⚠️ ADMIN_PASSWORD 未设置，Web 控制台无需鉴权\n")
	}

	r := chi.NewRouter()
	webAuth := api.WebAuthMiddleware(cfg.AdminPassword)

	// 登录页
	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, app.ResolveProjectPath("web/login.html"))
	})
	r.Post("/login", func(w http.ResponseWriter, req *http.Request) {
		pass := req.FormValue("password")
		if cfg.AdminPassword == "" || pass == cfg.AdminPassword {
			api.SetSessionCookie(w, cfg.AdminPassword)
			next := req.FormValue("next")
			if next == "" {
				next = "/dashboard"
			}
			http.Redirect(w, req, next, http.StatusFound)
		} else {
			http.Redirect(w, req, "/login?error=1&next="+req.FormValue("next"), http.StatusFound)
		}
	})

	// 退出登录
	r.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		api.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	// 静态文件服务（需要鉴权）
	fs := http.FileServer(http.Dir(app.ResolveProjectPath("web")))
	r.With(webAuth).Handle("/web/*", http.StripPrefix("/web/", fs))
	rulesFS := http.FileServer(http.Dir(app.ResolveProjectPath("rules")))
	r.With(webAuth).Handle("/rules/*", http.StripPrefix("/rules/", rulesFS))

	// 页面路由（无 .html 后缀，需要鉴权）
	servePage := func(file string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, app.ResolveProjectPath(file))
		}
	}
	r.With(webAuth).Get("/dashboard", servePage("web/index.html"))
	r.With(webAuth).Get("/subscriptions", servePage("web/subscriptions.html"))
	r.With(webAuth).Get("/nodes", servePage("web/nodes.html"))
	r.With(webAuth).Get("/rule-sources", servePage("web/rule_sources.html"))
	r.With(webAuth).Get("/templates", servePage("web/templates.html"))
	r.With(webAuth).Get("/configs", servePage("web/configs.html"))
	r.With(webAuth).Get("/config-access-logs", servePage("web/config_access_logs.html"))

	// 根路径重定向到仪表盘
	r.With(webAuth).Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})

	// 公开接口（无需鉴权）
	r.Get("/subscribe", apiHandlers.GenerateConfig)
	r.Get("/health", apiHandlers.Health)
	r.Get("/rulesets/{name}", apiHandlers.ExportRuleSource)

	// API 路由（整体加鉴权）
	r.With(api.APIAuthMiddleware(cfg.AdminPassword)).Route("/api", func(r chi.Router) {
		r.Post("/cache/policies/clear", apiHandlers.ClearAllPolicyCache)

		// 订阅管理
		r.Get("/subscriptions", apiHandlers.ListSubscriptions)
		r.Post("/subscriptions", apiHandlers.CreateSubscription)
		r.Route("/subscriptions/{id}", func(r chi.Router) {
			r.Get("/", apiHandlers.GetSubscription)
			r.Put("/", apiHandlers.UpdateSubscription)
			r.Patch("/", apiHandlers.UpdateSubscription)
			r.Delete("/", apiHandlers.DeleteSubscription)
			r.Post("/sync", apiHandlers.SyncSubscription)
			r.Get("/sync-status", apiHandlers.GetSubscriptionSyncStatus)
		})

		// 模板管理
		r.Get("/templates", apiHandlers.ListTemplates)
		r.Post("/templates", apiHandlers.CreateTemplate)
		r.Route("/templates/{id}", func(r chi.Router) {
			r.Get("/", apiHandlers.GetTemplate)
			r.Put("/", apiHandlers.UpdateTemplate)
			r.Patch("/", apiHandlers.UpdateTemplate)
			r.Delete("/", apiHandlers.DeleteTemplate)
		})

		// 规则源管理
		r.Get("/rule-sources", apiHandlers.ListRuleSources)
		r.Post("/rule-sources", apiHandlers.CreateRuleSource)
		r.Route("/rule-sources/{id}", func(r chi.Router) {
			r.Get("/", apiHandlers.GetRuleSource)
			r.Put("/", apiHandlers.UpdateRuleSource)
			r.Patch("/", apiHandlers.UpdateRuleSource)
			r.Delete("/", apiHandlers.DeleteRuleSource)
			r.Post("/sync", apiHandlers.SyncRuleSource)
		})

		// 配置策略管理
		r.Get("/config-policies", apiHandlers.ListConfigPolicies)
		r.Get("/config-access-logs", apiHandlers.ListAllConfigAccessLogs)
		r.Post("/config-policies", apiHandlers.CreateConfigPolicy)
		r.Route("/config-policies/{id}", func(r chi.Router) {
			r.Get("/", apiHandlers.GetConfigPolicy)
			r.Put("/", apiHandlers.UpdateConfigPolicy)
			r.Patch("/", apiHandlers.UpdateConfigPolicy)
			r.Delete("/", apiHandlers.DeleteConfigPolicy)
			r.Delete("/cache", apiHandlers.ClearPolicyConfigCache)
			r.Get("/access-logs", apiHandlers.ListConfigPolicyAccessLogs)
		})

		// 节点管理
		r.Get("/nodes", apiHandlers.ListNodes)
		r.Post("/nodes", apiHandlers.CreateNode)
		r.Get("/nodes/stats", apiHandlers.GetNodeStats)
		r.Post("/nodes/import", apiHandlers.ImportNodes)
		r.Post("/nodes/batch", apiHandlers.BatchNodeOperation)
		r.Patch("/nodes/batch", apiHandlers.BatchNodeOperation)
		r.Route("/nodes/{id}", func(r chi.Router) {
			r.Get("/", apiHandlers.GetNode)
			r.Put("/", apiHandlers.UpdateNode)
			r.Patch("/", apiHandlers.UpdateNode)
			r.Delete("/", apiHandlers.DeleteNode)
		})
	})

	return r
}
