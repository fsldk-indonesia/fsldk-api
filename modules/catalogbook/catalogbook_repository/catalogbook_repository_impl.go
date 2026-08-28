package catalogbook_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/catalogbook/catalogbook_dto"
	"fsldk-api/modules/catalogbook/catalogbook_model"

	"gorm.io/gorm"
)

const selectCols = "b.bookID, b.bookSlug, b.isbn, b.bookTitle, b.authorName, " +
	"b.authorTypeID, at.authorTypeName, b.publisherName, " +
	"b.bookCategoryID, c.bookCategoryName, b.languageID, l.languageName, " +
	"b.availabilityTypeID, av.availabilityTypeName, " +
	"b.bookPdf, b.year, b.pages, b.description, b.synopsis, b.edition, " +
	"b.coverImage, b.favoriteCount, b.tags, b.metaKeywords, b.metaDescription, " +
	"b.isActive, b.createdDate"

// RepositoryImpl is the GORM-based Repository implementation.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository creates the Repository implementation.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) baseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("ms_catalog_book b").
		Joins("JOIN lk_book_category c ON c.bookCategoryID = b.bookCategoryID").
		Joins("JOIN lk_book_language l ON l.languageID = b.languageID").
		Joins("JOIN lk_book_author_type at ON at.authorTypeID = b.authorTypeID").
		Joins("JOIN lk_book_availability_type av ON av.availabilityTypeID = b.availabilityTypeID")
}

func (r *RepositoryImpl) List(ctx context.Context, f catalogbook_dto.Filter) ([]catalogbook_model.CatalogBook, int64, error) {
	q := r.baseQuery(ctx)
	if f.ActiveOnly {
		q = q.Where("b.isActive = 1")
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(b.bookTitle LIKE ? OR b.authorName LIKE ? OR b.publisherName LIKE ?)", like, like, like)
	}
	if len(f.BookCategoryIDs) > 0 {
		q = q.Where("b.bookCategoryID IN ?", f.BookCategoryIDs)
	}
	if len(f.AuthorTypeIDs) > 0 {
		q = q.Where("b.authorTypeID IN ?", f.AuthorTypeIDs)
	}
	if len(f.AvailabilityTypeIDs) > 0 {
		q = q.Where("b.availabilityTypeID IN ?", f.AvailabilityTypeIDs)
	}
	if len(f.LanguageIDs) > 0 {
		q = q.Where("b.languageID IN ?", f.LanguageIDs)
	}
	if len(f.Years) > 0 {
		q = q.Where("b.year IN ?", f.Years)
	}
	if f.Author != "" {
		q = q.Where("b.authorName LIKE ?", "%"+f.Author+"%")
	}
	if f.Publisher != "" {
		q = q.Where("b.publisherName LIKE ?", "%"+f.Publisher+"%")
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []catalogbook_model.CatalogBook
	err := q.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (catalogbook_model.CatalogBook, error) {
	var b catalogbook_model.CatalogBook
	err := r.baseQuery(ctx).Select(selectCols).Where(where, arg).Take(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return catalogbook_model.CatalogBook{}, ErrNotFound
	}
	return b, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (catalogbook_model.CatalogBook, error) {
	return r.findOne(ctx, "b.bookID = ?", id)
}

func (r *RepositoryImpl) FindBySlug(ctx context.Context, slug string) (catalogbook_model.CatalogBook, error) {
	return r.findOne(ctx, "b.bookSlug = ?", slug)
}

func (r *RepositoryImpl) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_catalog_book").
		Where("bookSlug = ? AND bookID <> ?", slug, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, b catalogbook_model.CatalogBook, actorID int64) (int64, error) {
	values := map[string]interface{}{
		"bookSlug":           b.BookSlug,
		"isbn":               b.ISBN,
		"bookTitle":          b.BookTitle,
		"authorName":         b.AuthorName,
		"authorTypeID":       b.AuthorTypeID,
		"publisherName":      b.PublisherName,
		"bookCategoryID":     b.BookCategoryID,
		"languageID":         b.LanguageID,
		"availabilityTypeID": b.AvailabilityTypeID,
		"bookPdf":            b.BookPdf,
		"year":               b.Year,
		"pages":              b.Pages,
		"description":        b.Description,
		"synopsis":           b.Synopsis,
		"edition":            b.Edition,
		"coverImage":         b.CoverImage,
		"tags":               b.Tags,
		"metaKeywords":       b.MetaKeywords,
		"metaDescription":    b.MetaDescription,
		"createdDate":        time.Now(),
		"createdBy":          actorID,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_catalog_book").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, b catalogbook_model.CatalogBook, actorID int64) error {
	return r.db.WithContext(ctx).Table("ms_catalog_book").Where("bookID = ?", id).Updates(map[string]interface{}{
		"isbn":               b.ISBN,
		"bookTitle":          b.BookTitle,
		"authorName":         b.AuthorName,
		"authorTypeID":       b.AuthorTypeID,
		"publisherName":      b.PublisherName,
		"bookCategoryID":     b.BookCategoryID,
		"languageID":         b.LanguageID,
		"availabilityTypeID": b.AvailabilityTypeID,
		"bookPdf":            b.BookPdf,
		"year":               b.Year,
		"pages":              b.Pages,
		"description":        b.Description,
		"synopsis":           b.Synopsis,
		"edition":            b.Edition,
		"coverImage":         b.CoverImage,
		"tags":               b.Tags,
		"metaKeywords":       b.MetaKeywords,
		"metaDescription":    b.MetaDescription,
		"bookSlug":           b.BookSlug,
		"updatedDate":        time.Now(),
		"updatedBy":          actorID,
	}).Error
}

func (r *RepositoryImpl) SetActive(ctx context.Context, id int64, isActive bool, actorID int64) error {
	return r.db.WithContext(ctx).Table("ms_catalog_book").Where("bookID = ?", id).Updates(map[string]interface{}{
		"isActive":    isActive,
		"updatedDate": time.Now(),
		"updatedBy":   actorID,
	}).Error
}

// IncrementFavorite bumps favoriteCount for active books only (the Laravel
// reference never checked isActive here — a small gap closed in this port).
// Returns ErrNotFound if the book does not exist or is inactive.
func (r *RepositoryImpl) IncrementFavorite(ctx context.Context, id int64) (int, error) {
	result := r.db.WithContext(ctx).Exec(
		"UPDATE ms_catalog_book SET favoriteCount = favoriteCount + 1 WHERE bookID = ? AND isActive = 1", id)
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return 0, ErrNotFound
	}
	var count int
	err := r.db.WithContext(ctx).Table("ms_catalog_book").Select("favoriteCount").Where("bookID = ?", id).Scan(&count).Error
	return count, err
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_catalog_book WHERE bookID = ?", id).Error
}

func (r *RepositoryImpl) Categories(ctx context.Context) ([]catalogbook_model.BookCategory, error) {
	var out []catalogbook_model.BookCategory
	err := r.db.WithContext(ctx).Table("lk_book_category").Order("bookCategoryName").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) Languages(ctx context.Context) ([]catalogbook_model.Language, error) {
	var out []catalogbook_model.Language
	err := r.db.WithContext(ctx).Table("lk_book_language").Order("languageName").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) AuthorTypes(ctx context.Context) ([]catalogbook_model.AuthorType, error) {
	var out []catalogbook_model.AuthorType
	err := r.db.WithContext(ctx).Table("lk_book_author_type").Order("authorTypeName").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) AvailabilityTypes(ctx context.Context) ([]catalogbook_model.AvailabilityType, error) {
	var out []catalogbook_model.AvailabilityType
	err := r.db.WithContext(ctx).Table("lk_book_availability_type").Order("availabilityTypeName").Find(&out).Error
	return out, err
}
