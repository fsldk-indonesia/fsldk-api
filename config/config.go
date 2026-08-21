// Package config memuat dan menyediakan konfigurasi aplikasi.
// Seluruh nilai yang bergantung lingkungan dibaca dari environment variable
// (berkas app.env pada pengembangan) mengikuti prinsip 12-Factor App.
package config

import (
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig menyimpan seluruh konfigurasi aplikasi FSLDK API.
type AppConfig struct {
	AppEnv      string `mapstructure:"APP_ENV"`
	AppHost     string `mapstructure:"APP_HOST"`
	AppPort     string `mapstructure:"APP_PORT"`
	AppURL      string `mapstructure:"APP_URL"`
	FrontendURL string `mapstructure:"FRONTEND_URL"`
	Timezone    string `mapstructure:"TZ"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`

	DBHost        string `mapstructure:"DB_HOST"`
	DBPort        string `mapstructure:"DB_PORT"`
	DBName        string `mapstructure:"DB_NAME"`
	DBUser        string `mapstructure:"DB_USER"`
	DBPassword    string `mapstructure:"DB_PASSWORD"`
	DBMaxOpenConn int    `mapstructure:"DB_MAX_OPEN_CONN"`
	DBMaxIdleConn int    `mapstructure:"DB_MAX_IDLE_CONN"`

	JWTSecret               string `mapstructure:"JWT_SECRET"`
	JWTRefreshSecret        string `mapstructure:"JWT_REFRESH_SECRET"`
	JWTAccessExpireMinutes  int    `mapstructure:"JWT_ACCESS_EXPIRE_MINUTES"`
	JWTRefreshExpireMinutes int    `mapstructure:"JWT_REFRESH_EXPIRE_MINUTES"`

	GoogleClientID       string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleAllowedDomains string `mapstructure:"GOOGLE_ALLOWED_DOMAINS"`
	GoogleDefaultRole    string `mapstructure:"GOOGLE_DEFAULT_ROLE"`
	GoogleTokenInfoURL   string `mapstructure:"GOOGLE_TOKENINFO_URL"`

	EmailVerificationExpireMinutes int    `mapstructure:"EMAIL_VERIFICATION_EXPIRE_MINUTES"`
	PasswordResetExpireMinutes     int    `mapstructure:"PASSWORD_RESET_EXPIRE_MINUTES"`
	RegisterDefaultRole            string `mapstructure:"REGISTER_DEFAULT_ROLE"`

	SMTPHost        string `mapstructure:"SMTP_HOST"`
	SMTPPort        int    `mapstructure:"SMTP_PORT"`
	SMTPUsername    string `mapstructure:"SMTP_USERNAME"`
	SMTPPassword    string `mapstructure:"SMTP_PASSWORD"`
	MailFromAddress string `mapstructure:"MAIL_FROM_ADDRESS"`
	MailFromName    string `mapstructure:"MAIL_FROM_NAME"`

	CorsAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`

	GiphyAPIKey string `mapstructure:"GIPHY_API_KEY"` // GIF/sticker picker for comment_service; empty = feature returns empty results, not an error

	KirimdevAPIKey             string `mapstructure:"KIRIMDEV_API_KEY"`
	KirimdevPhoneNumberID      string `mapstructure:"KIRIMDEV_PHONE_NUMBER_ID"`
	KirimdevBaseURL            string `mapstructure:"KIRIMDEV_BASE_URL"`
	KirimdevTemplateLanguage   string `mapstructure:"KIRIMDEV_TEMPLATE_LANGUAGE"`
	KirimdevWebhookSecretsRaw  string `mapstructure:"KIRIMDEV_WEBHOOK_SECRETS"`      // comma-separated, lihat KirimdevWebhookSecrets()
	KirimdevReplyWindowMinutes int    `mapstructure:"KIRIMDEV_REPLY_WINDOW_MINUTES"` // toleransi replay signature webhook (§7 techspec)

	// Job queue (modules/jobqueue, §1b techspec) — dipakai shortlinkrequest_service
	// untuk kirim WhatsApp/email asinkron dengan retry, bukan lagi goroutine langsung.
	JobQueueWorkerCount           int     `mapstructure:"JOBQUEUE_WORKER_COUNT"`
	JobQueuePollIntervalMS        int     `mapstructure:"JOBQUEUE_POLL_INTERVAL_MS"`
	JobQueueStuckThresholdMinutes int     `mapstructure:"JOBQUEUE_STUCK_THRESHOLD_MINUTES"`
	JobQueueDefaultMaxAttempts    int     `mapstructure:"JOBQUEUE_DEFAULT_MAX_ATTEMPTS"`
	JobQueueBackoffScheduleRaw    string  `mapstructure:"JOBQUEUE_BACKOFF_SCHEDULE_SECONDS"` // comma-separated detik, mis. "15,30,60"
	JobQueueWhatsAppRatePerMinute float64 `mapstructure:"JOBQUEUE_WHATSAPP_RATE_PER_MINUTE"`
}

// JobQueueBackoffSchedule mem-parsing JOBQUEUE_BACKOFF_SCHEDULE_SECONDS
// ("15,30,60") menjadi []time.Duration. Entri yang tidak valid dilewati.
func (c AppConfig) JobQueueBackoffSchedule() []time.Duration {
	parts := strings.Split(c.JobQueueBackoffScheduleRaw, ",")
	out := make([]time.Duration, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if secs, err := strconv.Atoi(v); err == nil {
			out = append(out, time.Duration(secs)*time.Second)
		}
	}
	return out
}

// KirimdevWebhookSecrets mengembalikan daftar secret webhook Kirimdev yang
// valid (comma-separated di env) — mendukung multi-value agar rotasi secret
// tidak menyebabkan downtime (secret lama & baru sama-sama valid sementara).
func (c AppConfig) KirimdevWebhookSecrets() []string {
	parts := strings.Split(c.KirimdevWebhookSecretsRaw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// AllowedGoogleDomains mengembalikan daftar domain email yang diizinkan login Google.
// Slice kosong berarti seluruh domain diizinkan.
func (c AppConfig) AllowedGoogleDomains() []string {
	if strings.TrimSpace(c.GoogleAllowedDomains) == "" {
		return nil
	}
	parts := strings.Split(c.GoogleAllowedDomains, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, strings.ToLower(v))
		}
	}
	return out
}

// AllowedCorsOrigins mengembalikan daftar origin yang diizinkan CORS.
func (c AppConfig) AllowedCorsOrigins() []string {
	parts := strings.Split(c.CorsAllowedOrigins, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// Load membaca konfigurasi dari berkas app.env dan environment variable.
func Load() (AppConfig, error) {
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	setDefaults()

	// Berkas app.env bersifat opsional bila seluruh nilai disediakan via environment.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return AppConfig{}, err
		}
	}

	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return AppConfig{}, err
	}
	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_HOST", "0.0.0.0")
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("APP_URL", "http://localhost:8080")
	viper.SetDefault("FRONTEND_URL", "http://localhost:4200")
	viper.SetDefault("TZ", "Asia/Jakarta")
	viper.SetDefault("LOG_LEVEL", "info")

	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("DB_NAME", "fsldk_db")
	viper.SetDefault("DB_USER", "root")
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("DB_MAX_OPEN_CONN", 50)
	viper.SetDefault("DB_MAX_IDLE_CONN", 10)

	viper.SetDefault("JWT_ACCESS_EXPIRE_MINUTES", 60)
	viper.SetDefault("JWT_REFRESH_EXPIRE_MINUTES", 43200)

	viper.SetDefault("GOOGLE_DEFAULT_ROLE", "Kontributor")
	viper.SetDefault("GOOGLE_TOKENINFO_URL", "https://oauth2.googleapis.com/tokeninfo")
	viper.SetDefault("REGISTER_DEFAULT_ROLE", "Kontributor")
	viper.SetDefault("EMAIL_VERIFICATION_EXPIRE_MINUTES", 60)
	viper.SetDefault("PASSWORD_RESET_EXPIRE_MINUTES", 60)

	viper.SetDefault("SMTP_PORT", 587)
	viper.SetDefault("MAIL_FROM_ADDRESS", "no-reply@fsldk-indonesia.com")
	viper.SetDefault("MAIL_FROM_NAME", "FSLDK Indonesia")

	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:4200")

	viper.SetDefault("KIRIMDEV_BASE_URL", "https://api.kirimdev.com/v1")
	viper.SetDefault("KIRIMDEV_TEMPLATE_LANGUAGE", "id")
	viper.SetDefault("KIRIMDEV_REPLY_WINDOW_MINUTES", 5)

	viper.SetDefault("JOBQUEUE_WORKER_COUNT", 2)
	viper.SetDefault("JOBQUEUE_POLL_INTERVAL_MS", 1500)
	viper.SetDefault("JOBQUEUE_STUCK_THRESHOLD_MINUTES", 10)
	viper.SetDefault("JOBQUEUE_DEFAULT_MAX_ATTEMPTS", 5)
	viper.SetDefault("JOBQUEUE_BACKOFF_SCHEDULE_SECONDS", "15,30,60")
	viper.SetDefault("JOBQUEUE_WHATSAPP_RATE_PER_MINUTE", 8)
}
