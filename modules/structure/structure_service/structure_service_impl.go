package structure_service

import (
	"context"
	"os"
	"path/filepath"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/modules/structure/structure_dto"
	"fsldk-api/modules/structure/structure_model"
	"fsldk-api/modules/structure/structure_repository"
)

// sortColumns is the CMS list sort whitelist — see structure_dto.Filter.OrderBy.
var sortColumns = map[string]string{
	"batch":         "batch",
	"period":        "period",
	"structureName": "structureName",
	"createdDate":   "createdDate",
}

type serviceImpl struct {
	repo structure_repository.Repository
}

// NewService creates a new structure service instance.
func NewService(repo structure_repository.Repository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) List(ctx context.Context, f structure_dto.Filter) ([]structure_model.Structure, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *serviceImpl) CMSList(ctx context.Context, q dto.ListQuery, f structure_dto.Filter) ([]structure_model.Structure, int64, error) {
	f.Limit = q.Limit
	f.Offset = q.Offset()
	f.OrderBy = q.OrderBy(sortColumns, "createdDate DESC")
	if f.Search == "" {
		f.Search = q.Search
	}
	return s.repo.List(ctx, f)
}

func (s *serviceImpl) GetByID(ctx context.Context, id int64) (structure_model.Structure, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *serviceImpl) Create(ctx context.Context, req structure_dto.CreateRequest, authorID int64) (int64, error) {
	model := structure_model.Structure{
		Batch:                req.Batch,
		Period:               req.Period,
		StructureName:        req.StructureName,
		StructureDescription: req.StructureDescription,
		LogoImage:            &req.LogoImage,
		StructureImage:       &req.StructureImage,
	}
	return s.repo.Create(ctx, model, authorID)
}

func (s *serviceImpl) Update(ctx context.Context, id int64, req structure_dto.UpdateRequest, updatedBy int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	model := structure_model.Structure{
		Batch:                req.Batch,
		Period:               req.Period,
		StructureName:        req.StructureName,
		StructureDescription: req.StructureDescription,
	}

	if req.LogoImage != nil {
		model.LogoImage = req.LogoImage
		if existing.LogoImage != nil && *existing.LogoImage != *req.LogoImage {
			_ = os.Remove(filepath.Join("assets", *existing.LogoImage))
		}
	}

	if req.StructureImage != nil {
		model.StructureImage = req.StructureImage
		if existing.StructureImage != nil && *existing.StructureImage != *req.StructureImage {
			_ = os.Remove(filepath.Join("assets", *existing.StructureImage))
		}
	}

	return s.repo.Update(ctx, id, model, updatedBy)
}

func (s *serviceImpl) Delete(ctx context.Context, id int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	if existing.LogoImage != nil {
		_ = os.Remove(filepath.Join("assets", *existing.LogoImage))
	}
	if existing.StructureImage != nil {
		_ = os.Remove(filepath.Join("assets", *existing.StructureImage))
	}

	return nil
}

// BulkDelete deletes multiple structures, reusing Delete's validation/file-
// cleanup per ID. Best-effort like the CMS bulk-delete pattern elsewhere
// (news, event, comment, schedule): an ID already gone is silently skipped.
func (s *serviceImpl) BulkDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return apperror.BadRequest("Tidak ada struktur yang dipilih")
	}
	for _, id := range ids {
		_ = s.Delete(ctx, id)
	}
	return nil
}
