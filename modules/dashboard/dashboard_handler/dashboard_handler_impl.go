package dashboard_handler

import (
	"strconv"

	"fsldk-api/base/httphelper"
	"fsldk-api/modules/dashboard/dashboard_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc dashboard_service.Service }

// NewHandler membuat Handler dashboard.
func NewHandler(svc dashboard_service.Service) Handler { return &HandlerImpl{svc: svc} }

func (h *HandlerImpl) Summary(c *gin.Context) {
	data, err := h.svc.Summary(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) Recent(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	data, err := h.svc.Recent(c.Request.Context(), limit)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}
