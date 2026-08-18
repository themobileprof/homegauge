package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/homegauge/homegauge/backend/internal/admin"
	"github.com/homegauge/homegauge/backend/internal/ai"
	"github.com/homegauge/homegauge/backend/internal/applications"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/calculator"
	"github.com/homegauge/homegauge/backend/internal/config"
	"github.com/homegauge/homegauge/backend/internal/countries"
	"github.com/homegauge/homegauge/backend/internal/documents"
	"github.com/homegauge/homegauge/backend/internal/eligibility"
	"github.com/homegauge/homegauge/backend/internal/middleware"
	"github.com/homegauge/homegauge/backend/internal/mortgages"
	"github.com/homegauge/homegauge/backend/internal/platform/db"
	"github.com/homegauge/homegauge/backend/internal/platform/mailer"
	"github.com/homegauge/homegauge/backend/internal/platform/migrate"
	redisx "github.com/homegauge/homegauge/backend/internal/platform/redis"
	"github.com/homegauge/homegauge/backend/internal/platform/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	sqlDB, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	wd, _ := os.Getwd()
	migDir, err := migrate.FindDir(wd)
	if err != nil {
		slog.Error("migrations", "err", err)
		os.Exit(1)
	}
	if err := migrate.Up(sqlDB, migDir); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	rdb, err := redisx.Connect(cfg.RedisURL)
	if err != nil {
		slog.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	docRoot := getenv("DOCUMENT_STORAGE_PATH", filepath.Join(os.TempDir(), "homegauge-docs"))
	store, err := storage.NewLocalStore(docRoot, cfg.SessionSecret)
	if err != nil {
		slog.Error("storage", "err", err)
		os.Exit(1)
	}

	mail := mailer.LogMailer{From: cfg.SMTPFrom}
	authSvc := auth.NewService(sqlDB, rdb, mail, cfg.AppURL, cfg.SessionTTL)
	authHandler := auth.NewHandler(authSvc)
	mortgageSvc := mortgages.NewService(sqlDB)
	mortgageHandler := mortgages.NewHandler(mortgageSvc)
	countrySvc := countries.NewService(sqlDB)
	countryHandler := countries.NewHandler(countrySvc)
	eligSvc := eligibility.NewService(sqlDB, mortgageSvc, cfg.DefaultITIPct)
	eligHandler := eligibility.NewHandler(eligSvc)
	docSvc := documents.NewService(sqlDB, store)
	docHandler := documents.NewHandler(docSvc)
	aiClient := ai.NewClient(
		cfg.AnthropicAPIKey, cfg.AnthropicModel,
		cfg.GeminiAPIKey, cfg.GeminiModel,
		cfg.DeepSeekAPIKey, cfg.DeepSeekModel,
	)
	slog.Info("ai reserved for unstructured jobs", "routing", aiClient.JobRouting(), "configured", aiClient.ConfiguredProviders())
	appSvc := applications.NewService(sqlDB)
	appHandler := applications.NewHandler(appSvc)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), middleware.CORS(cfg.CORSOrigins))
	r.MaxMultipartMemory = 12 << 20

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "homegauge-api",
			"brand":   "HomeGauge",
		})
	})

	api := r.Group("/api/v1")
	authHandler.RegisterRoutes(api.Group("/auth"))
	authHandler.RegisterAuthenticated(api.Group("/auth", middleware.Authenticate(authSvc)))
	calculator.NewHandler().RegisterRoutes(api.Group("/calculator"))
	countryHandler.RegisterRoutes(api)
	mortgageHandler.RegisterRoutes(api)
	docHandler.RegisterPublicSigned(api)

	authed := api.Group("", middleware.Authenticate(authSvc))
	eligHandler.RegisterRoutes(authed)
	docHandler.RegisterCustomer(authed)
	appHandler.RegisterCustomer(authed)

	adminAPI := api.Group("/admin", middleware.Authenticate(authSvc), middleware.RequireRoles(auth.RoleAdmin))
	adminAPI.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "scope": "admin"})
	})
	adminAPI.GET("/ai-status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"configured": aiClient.ConfiguredProviders(),
			"routing":    aiClient.JobRouting(),
			"policy":     "AI only for unstructured work (e.g. salary statement extraction). Eligibility, affordability, readiness, and advisor checklists are programmatic.",
		})
	})
	adminHandler := admin.NewHandler(sqlDB, aiClient)
	adminHandler.Register(adminAPI)
	appHandler.RegisterAdmin(adminAPI)

	advisor := api.Group("/advisor", middleware.Authenticate(authSvc), middleware.RequireRoles(auth.RoleAdvisor))
	advisor.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "scope": "advisor"})
	})
	appHandler.RegisterAdvisor(advisor)
	docHandler.RegisterStaff(advisor)

	slog.Info("homegauge api listening", "addr", cfg.APIAddr, "docs", docRoot)
	if err := r.Run(cfg.APIAddr); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
