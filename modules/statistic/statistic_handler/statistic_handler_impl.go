package statistic_handler

import (
	"fsldk-api/base/dto"
	"fsldk-api/base/httphelper"
	"fsldk-api/modules/statistic/statistic_service"

	"github.com/gin-gonic/gin"
)

type handlerImpl struct {
	svc statistic_service.Service
}

// NewHandler creates a new instance of statistic Handler.
func NewHandler(svc statistic_service.Service) Handler {
	return &handlerImpl{svc: svc}
}

func (h *handlerImpl) NetworkStats(c *gin.Context) {
	resp, err := h.svc.NetworkStats(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berhasil mengambil statistik jaringan", resp)
}

func (h *handlerImpl) Directory(c *gin.Context) {
	q := dto.ParseListQuery(c)
	items, total, err := h.svc.Directory(c.Request.Context(), q, c.Query("type"), c.Query("province"))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Berhasil mengambil direktori jaringan", httphelper.BuildPagination(c, items, total, q.Page, q.Limit))
}
