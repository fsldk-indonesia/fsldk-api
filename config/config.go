// Package config memuat dan menyediakan konfigurasi aplikasi.
// Seluruh nilai yang bergantung lingkungan dibaca dari environment variable
// (berkas app.env pada pengembangan) mengikuti prinsip 12-Factor App.
package config

import (
	"strings"

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
	viper.SetDefault("REGISTER_DEFAULT_ROLE", "Kontributor")
	viper.SetDefault("EMAIL_VERIFICATION_EXPIRE_MINUTES", 60)
	viper.SetDefault("PASSWORD_RESET_EXPIRE_MINUTES", 60)

	viper.SetDefault("SMTP_PORT", 587)
	viper.SetDefault("MAIL_FROM_ADDRESS", "no-reply@fsldk-indonesia.com")
	viper.SetDefault("MAIL_FROM_NAME", "FSLDK Indonesia")

	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:4200")
}
