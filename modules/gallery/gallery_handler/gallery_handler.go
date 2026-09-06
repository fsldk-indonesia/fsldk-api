package gallery_handler

import "github.com/gin-gonic/gin"

// Handler defines HTTP handlers for the gallery module.
type Handler interface {
	// Public endpoints
	ListPublic(c *gin.Context)
	ShowPublic(c *gin.Context)
	ListPhotosPublic(c *gin.Context)

	// CMS endpoints
	ListCMS(c *gin.Context)
	ShowCMS(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)

	// CMS photo endpoints
	ListPhotosCMS(c *gin.Context)
	AddPhoto(c *gin.Context)
	UpdatePhoto(c *gin.Context)
	DeletePhoto(c *gin.Context)
	ReorderPhotos(c *gin.Context)
}
