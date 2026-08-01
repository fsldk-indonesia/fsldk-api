package middlewares

import (
	"time"

	"fsldk-api/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS mengonfigurasi Cross-Origin Resource Sharing berdasarkan whitelist origin.
func CORS(cfg config.AppConfig) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedCorsOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
