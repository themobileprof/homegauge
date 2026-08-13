package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/calculator"
	"github.com/homegauge/homegauge/backend/internal/config"
	"github.com/homegauge/homegauge/backend/internal/middleware"
	"github.com/homegauge/homegauge/backend/internal/platform/db"
	"github.com/homegauge/homegauge/backend/internal/platform/mailer"
	"github.com/homegauge/homegauge/backend/internal/platform/migrate"
	redisx "github.com/homegauge/homegauge/backend/internal/platform/redis"
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

	mail := mailer.LogMailer{From: cfg.SMTPFrom}
	authSvc := auth.NewService(sqlDB, rdb, mail, cfg.AppURL, cfg.SessionTTL)
	authHandler := auth.NewHandler(authSvc)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), middleware.CORS(cfg.CORSOrigins))

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

	admin := api.Group("/admin", middleware.Authenticate(authSvc), middleware.RequireRoles(auth.RoleAdmin))
	admin.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "scope": "admin"})
	})

	advisor := api.Group("/advisor", middleware.Authenticate(authSvc), middleware.RequireRoles(auth.RoleAdvisor, auth.RoleAdmin))
	advisor.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "scope": "advisor"})
	})

	slog.Info("homegauge api listening", "addr", cfg.APIAddr)
	if err := r.Run(cfg.APIAddr); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
