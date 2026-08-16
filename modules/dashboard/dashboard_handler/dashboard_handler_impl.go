package dashboard_handler

import (
	"fsldk-api/base/appctx"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/dashboard/dashboard_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc dashboard_service.Service }

// NewHandler membuat Handler dashboard.
func NewHandler(svc dashboard_service.Service) Handler { return &HandlerImpl{svc: svc} }

func (h *HandlerImpl) Summary(c *gin.Context) {
	caller := dashboard_service.CallerScope{
		OrganizationID:       appctx.OrganizationID(c),
		OrganizationTypeCode: appctx.OrganizationTypeCode(c),
		WildcardTierAccess:   appctx.WildcardTierAccess(c),
	}
	data, err := h.svc.Summary(c.Request.Context(), caller)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}
