package subscription_handler

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/subscription/subscription_dto"
	"fsldk-api/modules/subscription/subscription_service"

	"github.com/gin-gonic/gin"
)

type handlerImpl struct {
	svc subscription_service.Service
}

// NewHandler creates a new instance of subscription Handler.
func NewHandler(svc subscription_service.Service) Handler {
	return &handlerImpl{svc: svc}
}

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func (h *handlerImpl) Subscribe(c *gin.Context) {
	var req subscription_dto.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	isResubscribe, err := h.svc.Subscribe(c.Request.Context(), req.Email)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	message := "Terima kasih! Email kamu berhasil didaftarkan untuk berlangganan kabar FSLDK."
	if isResubscribe {
		message = "Selamat datang kembali! Email kamu berhasil didaftarkan kembali."
	}
	httphelper.Created(c, message, nil)
}

func (h *handlerImpl) Unsubscribe(c *gin.Context) {
	var req subscription_dto.UnsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	if err := h.svc.Unsubscribe(c.Request.Context(), req.Email, req.Token); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Anda berhasil berhenti berlangganan", nil)
}

func (h *handlerImpl) List(c *gin.Context) {
	q := dto.ParseListQuery(c)

	var isActive *bool
	if v := c.Query("isActive"); v != "" {
		b := v == "true" || v == "1"
		isActive = &b
	}
	from := c.Query("from")
	to := c.Query("to")

	items, total, err := h.svc.List(c.Request.Context(), q, isActive, from, to)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berhasil mengambil daftar subscriber", httphelper.BuildPagination(c, items, total, q.Page, q.Limit))
}

func (h *handlerImpl) Get(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	detail, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berhasil mengambil detail subscriber", detail)
}

func (h *handlerImpl) BulkAdd(c *gin.Context) {
	var req subscription_dto.BulkAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	result, err := h.svc.BulkAdd(c.Request.Context(), req.Emails)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berhasil memproses daftar email", result)
}

func (h *handlerImpl) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req subscription_dto.UpdateSubscriberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	detail, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Subscriber berhasil diperbarui", detail)
}

func (h *handlerImpl) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Subscriber berhasil dihapus", nil)
}

func (h *handlerImpl) BulkDelete(c *gin.Context) {
	var req subscription_dto.BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	if err := h.svc.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Subscriber terpilih berhasil dihapus", nil)
}
