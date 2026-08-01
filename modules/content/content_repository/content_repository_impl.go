package content_repository

import (
	"context"
	"database/sql"
	"errors"

	"fsldk-api/modules/content/content_dto"
	"fsldk-api/modules/content/content_model"

	"github.com/jmoiron/sqlx"
)

// RepositoryImpl adalah implementasi Repository berbasis sqlx.
type RepositoryImpl struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) ListContent(ctx context.Context, activeOnly bool) ([]content_model.Content, error) {
	q := `SELECT contentID, contentKey, contentTitle, contentBody, contentType, sortOrder, isActive FROM ms_cms_content`
	if activeOnly {
		q += " WHERE isActive = 1"
	}
	q += " ORDER BY sortOrder, contentKey"
	var out []content_model.Content
	err := r.db.SelectContext(ctx, &out, q)
	return out, err
}

func (r *RepositoryImpl) GetContentByKey(ctx context.Context, key string) (content_model.Content, error) {
	var c content_model.Content
	err := r.db.GetContext(ctx, &c,
		`SELECT contentID, contentKey, contentTitle, contentBody, contentType, sortOrder, isActive
		 FROM ms_cms_content WHERE contentKey = ? LIMIT 1`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return content_model.Content{}, ErrNotFound
	}
	return c, err
}

func (r *RepositoryImpl) UpdateContent(ctx context.Context, key, title, body string, updatedBy int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE ms_cms_content SET contentTitle = ?, contentBody = ?, updatedDate = NOW(), updatedBy = ? WHERE contentKey = ?`,
		title, body, updatedBy, key)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepositoryImpl) ListOrg(ctx context.Context, activeOnly bool) ([]content_model.OrgMember, error) {
	q := `SELECT structureID, memberName, position, photoURL, level, sortOrder, isActive FROM ms_organization_structure`
	if activeOnly {
		q += " WHERE isActive = 1"
	}
	q += " ORDER BY sortOrder, structureID"
	var out []content_model.OrgMember
	err := r.db.SelectContext(ctx, &out, q)
	return out, err
}

func (r *RepositoryImpl) CreateOrg(ctx context.Context, m content_dto.OrgRequest, createdBy int64) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ms_organization_structure (memberName, position, photoURL, level, sortOrder, isActive, createdDate, createdBy)
		 VALUES (?, ?, ?, ?, ?, 1, NOW(), ?)`,
		m.MemberName, m.Position, nullStr(m.PhotoURL), nullStr(m.Level), m.SortOrder, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *RepositoryImpl) UpdateOrg(ctx context.Context, id int64, m content_dto.OrgRequest) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE ms_organization_structure SET memberName = ?, position = ?, photoURL = ?, level = ?, sortOrder = ? WHERE structureID = ?`,
		m.MemberName, m.Position, nullStr(m.PhotoURL), nullStr(m.Level), m.SortOrder, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepositoryImpl) DeleteOrg(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_organization_structure WHERE structureID = ?", id)
	return err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
