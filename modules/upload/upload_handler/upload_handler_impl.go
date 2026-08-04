package upload_handler

import (
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/upload/upload_dto"
	"fsldk-api/modules/upload/upload_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc upload_service.Service }

// NewHandler membuat Handler upload.
func NewHandler(svc upload_service.Service) Handler { return &HandlerImpl{svc: svc} }

func (h *HandlerImpl) UploadImage(c *gin.Context) {
	fh, err := c.FormFile("image")
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("Berkas gambar wajib diunggah (field 'image')"))
		return
	}
	url, err := h.svc.SaveImage(fh)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	httphelper.Created(c, "Gambar berhasil diunggah", upload_dto.Response{URL: url})
}

func (h *HandlerImpl) UploadDocument(c *gin.Context) {
	fh, err := c.FormFile("document")
	if err != nil {
		httphelper.Error(c, apperror.BadRequest("Berkas dokumen wajib diunggah (field 'document')"))
		return
	}
	url, err := h.svc.SaveDocument(fh)
	if err != nil {
		httphelper.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	httphelper.Created(c, "Dokumen berhasil diunggah", upload_dto.Response{URL: url})
}
