package report_handler

import (
	"fmt"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/report/report_dto"
	"fsldk-api/modules/report/report_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc report_service.Service }

// NewHandler membuat Handler report.
func NewHandler(svc report_service.Service) Handler { return &HandlerImpl{svc: svc} }

func (h *HandlerImpl) ExportSubmissions(c *gin.Context) {
	formCode := c.Query("formCode")
	if formCode == "" {
		httphelper.Error(c, apperror.BadRequest("Parameter formCode wajib diisi"))
		return
	}
	format := c.DefaultQuery("format", "xlsx")

	caller := report_service.CallerScope{
		UserID:               appctx.UserID(c),
		OrganizationID:       appctx.OrganizationID(c),
		OrganizationTypeCode: appctx.OrganizationTypeCode(c),
		WildcardTierAccess:   appctx.WildcardTierAccess(c),
	}
	result, err := h.svc.ExportSubmissions(c.Request.Context(), caller, report_dto.ExportFilter{
		FormCode: formCode,
		Status:   c.Query("status"),
		Format:   format,
	})
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", result.FileName))
	c.Data(200, result.ContentType, result.Data)
}
