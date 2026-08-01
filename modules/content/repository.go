package content

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// ErrNotFound dikembalikan bila data tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository adalah kontrak akses data konten & struktur organisasi.
type Repository interface {
	ListContent(ctx context.Context, activeOnly bool) ([]Content, error)
	GetContentByKey(ctx context.Context, key string) (Content, error)
	UpdateContent(ctx context.Context, key, title, body string, updatedBy int64) error

	ListOrg(ctx context.Context, activeOnly bool) ([]OrgMember, error)
	CreateOrg(ctx context.Context, m OrgRequest, createdBy int64) (int64, error)
	UpdateOrg(ctx context.Context, id int64, m OrgRequest) error
	DeleteOrg(ctx context.Context, id int64) error
}

type repository struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &repository{db: db} }

func (r *repository) ListContent(ctx context.Context, activeOnly bool) ([]Content, error) {
	q := `SELECT contentID, contentKey, contentTitle, contentBody, contentType, sortOrder, isActive FROM ms_cms_content`
	if activeOnly {
		q += " WHERE isActive = 1"
	}
	q += " ORDER BY sortOrder, contentKey"
	var out []Content
	err := r.db.SelectContext(ctx, &out, q)
	return out, err
}

func (r *repository) GetContentByKey(ctx context.Context, key string) (Content, error) {
	var c Content
	err := r.db.GetContext(ctx, &c,
		`SELECT contentID, contentKey, contentTitle, contentBody, contentType, sortOrder, isActive
		 FROM ms_cms_content WHERE contentKey = ? LIMIT 1`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return Content{}, ErrNotFound
	}
	return c, err
}

func (r *repository) UpdateContent(ctx context.Context, key, title, body string, updatedBy int64) error {
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

func (r *repository) ListOrg(ctx context.Context, activeOnly bool) ([]OrgMember, error) {
	q := `SELECT structureID, memberName, position, photoURL, level, sortOrder, isActive FROM ms_organization_structure`
	if activeOnly {
		q += " WHERE isActive = 1"
	}
	q += " ORDER BY sortOrder, structureID"
	var out []OrgMember
	err := r.db.SelectContext(ctx, &out, q)
	return out, err
}

func (r *repository) CreateOrg(ctx context.Context, m OrgRequest, createdBy int64) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ms_organization_structure (memberName, position, photoURL, level, sortOrder, isActive, createdDate, createdBy)
		 VALUES (?, ?, ?, ?, ?, 1, NOW(), ?)`,
		m.MemberName, m.Position, nullStr(m.PhotoURL), nullStr(m.Level), m.SortOrder, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *repository) UpdateOrg(ctx context.Context, id int64, m OrgRequest) error {
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

func (r *repository) DeleteOrg(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_organization_structure WHERE structureID = ?", id)
	return err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
