package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv               string
	AppURL               string
	APIAddr              string
	DatabaseURL          string
	RedisURL             string
	SessionSecret        string
	SessionTTL           time.Duration
	CORSOrigins          []string
	SMTPFrom             string
	MailerMode           string
	AnthropicAPIKey      string
	AnthropicModel       string
	GeminiAPIKey         string
	GeminiModel          string
	DeepSeekAPIKey       string
	DeepSeekModel        string
	N8NWebhookURL        string
	S3Endpoint           string
	S3Region             string
	S3Bucket             string
	S3AccessKey          string
	S3SecretKey          string
	S3UsePathStyle       bool
	AutomationLevel      string
	SalaryVariancePct    float64
	SalaryPaydayLastDays int
	DefaultITIPct        float64
}

func Load() (Config, error) {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	cfg := Config{
		AppEnv:               getenv("APP_ENV", "development"),
		AppURL:               getenv("APP_URL", "http://localhost:3000"),
		APIAddr:              getenv("API_ADDR", ":8080"),
		DatabaseURL:          getenv("DATABASE_URL", "postgres://samuel@/homegauge?host=/var/run/postgresql"),
		RedisURL:             getenv("REDIS_URL", "redis://localhost:6379/0"),
		SessionSecret:        getenv("SESSION_SECRET", "dev-only-change-me-homegauge-session"),
		SessionTTL:           durationHours("SESSION_TTL_HOURS", 168),
		CORSOrigins:          splitCSV(getenv("CORS_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")),
		SMTPFrom:             getenv("SMTP_FROM", "noreply@homegauge.local"),
		MailerMode:           getenv("MAILER_MODE", "log"),
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:       getenv("ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
		GeminiAPIKey:         firstEnv("GEMINI_API_KEY", "GOOGLE_GENERATIVE_AI_API_KEY"),
		GeminiModel:          getenv("GEMINI_MODEL", "gemini-2.0-flash"),
		DeepSeekAPIKey:       os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekModel:        getenv("DEEPSEEK_MODEL", "deepseek-chat"),
		N8NWebhookURL:        os.Getenv("N8N_WEBHOOK_URL"),
		S3Endpoint:           getenv("S3_ENDPOINT", ""),
		S3Region:             getenv("S3_REGION", "auto"),
		S3Bucket:             getenv("S3_BUCKET", "homegauge-docs"),
		S3AccessKey:          os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:          os.Getenv("S3_SECRET_KEY"),
		S3UsePathStyle:       getenv("S3_USE_PATH_STYLE", "true") == "true",
		AutomationLevel:      getenv("AUTOMATION_LEVEL", "suggest_only"),
		SalaryVariancePct:    floatEnv("SALARY_VARIANCE_PCT", 15),
		SalaryPaydayLastDays: intEnv("SALARY_PAYDAY_LAST_DAYS", 7),
		DefaultITIPct:        floatEnv("DEFAULT_ITI_PCT", 35),
	}
	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func intEnv(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func floatEnv(k string, def float64) float64 {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return n
}

func durationHours(k string, def int) time.Duration {
	return time.Duration(intEnv(k, def)) * time.Hour
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
