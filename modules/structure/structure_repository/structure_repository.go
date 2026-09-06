package structure_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/structure/structure_dto"
	"fsldk-api/modules/structure/structure_model"
)

// ErrNotFound is returned when a structure cannot be found.
var ErrNotFound = errors.New("structure not found")

// Repository defines the data-access contract for structures.
type Repository interface {
	List(ctx context.Context, f structure_dto.Filter) ([]structure_model.Structure, int64, error)
	FindByID(ctx context.Context, id int64) (structure_model.Structure, error)
	Create(ctx context.Context, s structure_model.Structure, authorID int64) (int64, error)
	Update(ctx context.Context, id int64, s structure_model.Structure, updatedBy int64) error
	Delete(ctx context.Context, id int64) error
}
