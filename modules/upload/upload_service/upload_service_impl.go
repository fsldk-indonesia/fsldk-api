package upload_service

import (
	"mime/multipart"

	"fsldk-api/pkg/upload"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct{ uploader *upload.Uploader }

// NewService membuat Service upload.
func NewService(uploader *upload.Uploader) Service { return &ServiceImpl{uploader: uploader} }

func (s *ServiceImpl) SaveImage(fh *multipart.FileHeader) (string, error) {
	return s.uploader.SaveImage(fh)
}
