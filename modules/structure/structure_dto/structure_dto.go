package structure_dto

// CreateRequest is the body for creating a structure.
type CreateRequest struct {
	Batch                string `json:"batch" validate:"required,max=50"`
	Period               string `json:"period" validate:"required,max=50"`
	StructureName        string `json:"structureName" validate:"required,max=255"`
	StructureDescription string `json:"structureDescription" validate:"required"`
	LogoImage            string `json:"logoImage" validate:"required,max=255"`
	StructureImage       string `json:"structureImage" validate:"required,max=255"`
}

// UpdateRequest is the body for updating a structure.
type UpdateRequest struct {
	Batch                string  `json:"batch" validate:"required,max=50"`
	Period               string  `json:"period" validate:"required,max=50"`
	StructureName        string  `json:"structureName" validate:"required,max=255"`
	StructureDescription string  `json:"structureDescription" validate:"required"`
	LogoImage            *string `json:"logoImage" validate:"omitempty,max=255"`
	StructureImage       *string `json:"structureImage" validate:"omitempty,max=255"`
}

// Filter holds query parameters for listing structures (used by repository & service).
// OrderBy is a pre-built, whitelisted "column DIR" clause (see
// structure_service.sortColumns) — the repository never builds it from raw
// user input, avoiding the SQL-injection surface the old sort_by/sort_order
// query params previously opened up.
type Filter struct {
	Search   string
	DateFrom string // "YYYY-MM-DD", inclusive — filters createdDate
	DateTo   string // "YYYY-MM-DD", inclusive
	Limit    int
	Offset   int
	OrderBy  string
}

// BulkDeleteRequest is the body for deleting multiple structures at once.
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}
