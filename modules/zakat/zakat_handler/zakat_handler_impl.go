package zakat_handler

import (
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/zakat/zakat_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl is the Handler implementation.
type HandlerImpl struct{ svc zakat_service.Service }

// NewHandler builds the zakat Handler.
func NewHandler(svc zakat_service.Service) Handler { return &HandlerImpl{svc: svc} }

// GoldPrice serves GET /public/zakat/gold-price. ?refresh=1 forces an upstream
// re-fetch. Always 200 with the standard envelope — the frontend branches on
// the body's `success`, not the HTTP status.
func (h *HandlerImpl) GoldPrice(c *gin.Context) {
	force := c.Query("refresh") == "1"
	res := h.svc.GoldPrice(c.Request.Context(), force)
	httphelper.Success(c, "OK", res)
}
