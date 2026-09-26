package welcomepopup_handler

import (
	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
	"fsldk-api/modules/welcomepopup/welcomepopup_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc welcomepopup_service.Service }

// NewHandler membuat Handler welcomepopup.
func NewHandler(svc welcomepopup_service.Service) Handler { return &HandlerImpl{svc: svc} }

// Get melayani GET /welcome-popup (CMS, permission welcomepopup.view).
func (h *HandlerImpl) Get(c *gin.Context) {
	res, err := h.svc.Get(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

// Update melayani PUT /welcome-popup (CMS, permission welcomepopup.update).
func (h *HandlerImpl) Update(c *gin.Context) {
	var req welcomepopup_dto.UpdateRequest
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
	httphelper.Success(c, "Welcome Popup berhasil diperbarui", res)
}

// GetPublic melayani GET /public/welcome-popup — tanpa auth, tanpa
// permission check, dipanggil langsung dari halaman Beranda publik.
func (h *HandlerImpl) GetPublic(c *gin.Context) {
	res, err := h.svc.GetPublic(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}
