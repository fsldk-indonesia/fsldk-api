package donation_handler

import (
	"strconv"

	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc donation_service.Service }

// NewHandler membuat Handler donation.
func NewHandler(svc donation_service.Service) Handler { return &HandlerImpl{svc: svc} }

// optionalDonorUserID mengembalikan pointer userID bila pemanggil sedang
// login (lewat middlewares.OptionalAuth), atau nil untuk donasi tamu.
func optionalDonorUserID(c *gin.Context) *int64 {
	if uid := appctx.UserID(c); uid > 0 {
		return &uid
	}
	return nil
}

func (h *HandlerImpl) Create(c *gin.Context) {
	var req donation_dto.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.Create(c.Request.Context(), c.Param("slug"), optionalDonorUserID(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Donasi berhasil dibuat", data)
}

func (h *HandlerImpl) Detail(c *gin.Context) {
	data, err := h.svc.GetByPublicRef(c.Request.Context(), c.Param("publicRef"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

// Callback menerima webhook payment dari Bisabiller. Tidak memakai JWT
// (diamankan via signature/IP allowlist, lihat donation_service.ProcessCallback
// dan middlewares.IPAllowlist) — respons tetap dalam envelope httphelper
// standar seperti endpoint lain, cukup HTTP 200 untuk menandakan sukses.
func (h *HandlerImpl) Callback(c *gin.Context) {
	var req donation_dto.CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format callback tidak valid"))
		return
	}
	if err := h.svc.ProcessCallback(c.Request.Context(), req); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "OK", nil)
}

func (h *HandlerImpl) Status(c *gin.Context) {
	data, err := h.svc.Status(c.Request.Context(), c.Param("publicRef"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) PublicRecentDonations(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	data, err := h.svc.PublicRecentDonations(c.Request.Context(), c.Param("slug"), limit)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) MyList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	data, total, err := h.svc.MyList(c.Request.Context(), appctx.UserID(c), q)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func (h *HandlerImpl) CMSList(c *gin.Context) {
	q := dto.ParseListQuery(c)
	campaignID, _ := strconv.ParseInt(c.Query("campaignID"), 10, 64)
	data, total, err := h.svc.CMSList(c.Request.Context(), q, campaignID, c.Query("status"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", httphelper.BuildPagination(c, data, total, q.Page, q.Limit))
}

func donationIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID donasi tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *HandlerImpl) CMSGet(c *gin.Context) {
	id, ok := donationIDParam(c)
	if !ok {
		return
	}
	data, err := h.svc.CMSGet(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) AdminCreate(c *gin.Context) {
	var req donation_dto.AdminCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.AdminCreate(c.Request.Context(), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Created(c, "Donasi manual berhasil dicatat", data)
}

func (h *HandlerImpl) AdminUpdate(c *gin.Context) {
	id, ok := donationIDParam(c)
	if !ok {
		return
	}
	var req donation_dto.AdminUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.AdminUpdate(c.Request.Context(), id, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Donasi manual berhasil diperbarui", data)
}

func (h *HandlerImpl) AdminDelete(c *gin.Context) {
	id, ok := donationIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.AdminDelete(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Donasi manual berhasil dihapus", nil)
}
