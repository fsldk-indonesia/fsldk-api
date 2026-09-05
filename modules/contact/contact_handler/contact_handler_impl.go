package contact_handler

import (
	"strconv"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/contact/contact_dto"
	"fsldk-api/modules/contact/contact_service"

	"github.com/gin-gonic/gin"
)

type handlerImpl struct {
	svc contact_service.Service
}

// NewHandler creates a new instance of contact Handler.
func NewHandler(svc contact_service.Service) Handler {
	return &handlerImpl{svc: svc}
}

func (h *handlerImpl) Send(c *gin.Context) {
	var req contact_dto.SendContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}

	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}

	ip := c.ClientIP()
	if err := h.svc.Send(c.Request.Context(), req, ip); err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Created(c, "Pesan berhasil dikirim. Kami akan menindaklanjutinya secepatnya.", nil)
}

func (h *handlerImpl) List(c *gin.Context) {
	var q contact_dto.ContactListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httphelper.Error(c, apperror.BadRequest("Query parameter tidak valid"))
		return
	}

	res, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil daftar pesan", res)
}

func (h *handlerImpl) Show(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	detail, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Berhasil mengambil detail pesan", detail)
}

func (h *handlerImpl) MarkRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	if err := h.svc.MarkRead(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Pesan berhasil ditandai sudah dibaca", nil)
}

func (h *handlerImpl) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "Pesan berhasil dihapus", nil)
}
