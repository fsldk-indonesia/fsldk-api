// Package zakat_handler is the HTTP layer of the zakat module.
package zakat_handler

import "github.com/gin-gonic/gin"

// Handler is the zakat HTTP contract.
type Handler interface {
	GoldPrice(c *gin.Context)
}
