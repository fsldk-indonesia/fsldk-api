package structure_service

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/modules/structure/structure_dto"
	"fsldk-api/modules/structure/structure_model"
)

// Service defines the business logic for structures.
type Service interface {
	List(ctx context.Context, f structure_dto.Filter) ([]structure_model.Structure, int64, error)
	// CMSList builds the OrderBy clause from a whitelist (see sortColumns)
	// before delegating to List — the CMS endpoint is the only one driven by
	// user-controlled sort input, so the whitelisting lives here rather than
	// duplicated per caller.
	CMSList(ctx context.Context, q dto.ListQuery, f structure_dto.Filter) ([]structure_model.Structure, int64, error)
	GetByID(ctx context.Context, id int64) (structure_model.Structure, error)
	Create(ctx context.Context, req structure_dto.CreateRequest, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, req structure_dto.UpdateRequest, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
	// BulkDelete removes multiple structures by ID, best-effort per ID.
	BulkDelete(ctx context.Context, ids []int64) error
}
