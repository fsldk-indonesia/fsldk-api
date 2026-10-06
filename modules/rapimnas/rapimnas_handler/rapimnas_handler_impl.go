package rapimnas_handler

import (
	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/rapimnas/rapimnas_dto"
	"fsldk-api/modules/rapimnas/rapimnas_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc rapimnas_service.Service }

// NewHandler membuat Handler rapimnas.
func NewHandler(svc rapimnas_service.Service) Handler { return &HandlerImpl{svc: svc} }

// Get melayani GET /rapimnas-setup (CMS, permission rapimnas.view).
func (h *HandlerImpl) Get(c *gin.Context) {
	res, err := h.svc.Get(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

// Update melayani PUT /rapimnas-setup (CMS, permission rapimnas.update).
func (h *HandlerImpl) Update(c *gin.Context) {
	var req rapimnas_dto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	res, err := h.svc.Update(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Rapimnas Setup berhasil diperbarui", res)
}

// GetPublic melayani GET /public/rapimnas — tanpa auth, dipanggil langsung
// dari 6 halaman publik /rapimnas.
func (h *HandlerImpl) GetPublic(c *gin.Context) {
	res, err := h.svc.GetPublic(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}
