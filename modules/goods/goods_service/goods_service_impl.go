package goods_service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/microcosm-cc/bluemonday"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/ptr"
	"fsldk-api/base/slug"
	"fsldk-api/modules/goods/goods_dto"
	"fsldk-api/modules/goods/goods_model"
	"fsldk-api/modules/goods/goods_repository"
)

var sortColumns = map[string]string{
	"goodsName":   "g.goodsName",
	"price":       "g.price",
	"sortOrder":   "g.sortOrder",
	"createdDate": "g.createdDate",
}

// publicSortColumns are the fixed sort options exposed on the public catalog,
// distinct from the CMS's free-form whitelist.
var publicSortColumns = map[string]string{
	"newest":     "g.sortOrder ASC, g.publishedDate DESC",
	"name":       "g.goodsName ASC",
	"price_asc":  "g.price ASC",
	"price_desc": "g.price DESC",
	"featured":   "g.isFeatured DESC, g.sortOrder ASC",
}

// richTextPolicy sanitizes fullDescription (stored TinyMCE HTML) before it is
// persisted — no sanitizer exists elsewhere in this codebase for rich-text
// fields, so goods applies its own rather than trusting CMS input verbatim.
var richTextPolicy = bluemonday.UGCPolicy()

// FileDeleter is the narrow slice of pkg/upload.Uploader this service
// depends on — accepting an interface keeps goods_service decoupled from the
// upload package's implementation.
type FileDeleter interface {
	DeleteFile(publicURL string) error
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo   goods_repository.Repository
	upload FileDeleter
}

// NewService membuat Service goods.
func NewService(repo goods_repository.Repository, upload FileDeleter) Service {
	return &ServiceImpl{repo: repo, upload: upload}
}

func (s *ServiceImpl) PublicList(ctx context.Context, q dto.ListQuery, f goods_dto.Filter, sort string) ([]goods_model.Goods, int, error) {
	f.PublishedOnly = true
	f.Limit = q.Limit
	f.Offset = q.Offset()
	orderBy, ok := publicSortColumns[sort]
	if !ok {
		orderBy = publicSortColumns["newest"]
	}
	f.OrderBy = orderBy
	if f.Search == "" {
		f.Search = q.Search
	}
	return s.list(ctx, f)
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, f goods_dto.Filter) ([]goods_model.Goods, int, error) {
	f.PublishedOnly = false
	f.Limit = q.Limit
	f.Offset = q.Offset()
	f.OrderBy = q.OrderBy(sortColumns, "g.createdDate DESC")
	if f.Search == "" {
		f.Search = q.Search
	}
	return s.list(ctx, f)
}

func (s *ServiceImpl) list(ctx context.Context, f goods_dto.Filter) ([]goods_model.Goods, int, error) {
	data, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	if data == nil {
		data = []goods_model.Goods{}
	}
	return data, int(total), nil
}

func (s *ServiceImpl) detail(ctx context.Context, g goods_model.Goods) (goods_dto.DetailResponse, error) {
	images, err := s.repo.ListImages(ctx, g.GoodsID)
	if err != nil {
		return goods_dto.DetailResponse{}, apperror.Internal("")
	}
	urls := make([]string, len(images))
	for i, img := range images {
		urls[i] = img.ImageUrl
	}
	return goods_dto.DetailResponse{Goods: g, Images: urls}, nil
}

func (s *ServiceImpl) PublicDetail(ctx context.Context, slugStr string) (goods_dto.DetailResponse, error) {
	g, err := s.repo.FindBySlug(ctx, slugStr)
	if err != nil || !g.IsPublished {
		return goods_dto.DetailResponse{}, apperror.NotFound("Produk tidak ditemukan")
	}
	return s.detail(ctx, g)
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (goods_dto.DetailResponse, error) {
	g, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return goods_dto.DetailResponse{}, apperror.NotFound("Produk tidak ditemukan")
	}
	return s.detail(ctx, g)
}

func (s *ServiceImpl) Create(ctx context.Context, req goods_dto.Request, actorID int64) (goods_dto.DetailResponse, error) {
	if err := s.validateRequest(ctx, req, 0); err != nil {
		return goods_dto.DetailResponse{}, err
	}
	entity := s.fromRequest(req)
	slugStr, err := s.uniqueSlug(ctx, req.GoodsName, 0)
	if err != nil {
		return goods_dto.DetailResponse{}, err
	}
	entity.GoodsSlug = slugStr
	id, err := s.repo.Create(ctx, entity, actorID)
	if err != nil {
		return goods_dto.DetailResponse{}, apperror.Internal("Gagal menyimpan produk")
	}
	if err := s.repo.ReplaceImages(ctx, id, req.ImageUrls); err != nil {
		return goods_dto.DetailResponse{}, apperror.Internal("Gagal menyimpan gambar produk")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req goods_dto.Request, actorID int64) (goods_dto.DetailResponse, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return goods_dto.DetailResponse{}, apperror.NotFound("Produk tidak ditemukan")
	}
	if err := s.validateRequest(ctx, req, id); err != nil {
		return goods_dto.DetailResponse{}, err
	}
	entity := s.fromRequest(req)
	slugStr := existing.GoodsSlug
	if existing.GoodsName != req.GoodsName {
		slugStr, err = s.uniqueSlug(ctx, req.GoodsName, id)
		if err != nil {
			return goods_dto.DetailResponse{}, err
		}
	}
	entity.GoodsSlug = slugStr
	if err := s.repo.Update(ctx, id, entity, actorID); err != nil {
		return goods_dto.DetailResponse{}, apperror.Internal("")
	}
	if err := s.repo.ReplaceImages(ctx, id, req.ImageUrls); err != nil {
		return goods_dto.DetailResponse{}, apperror.Internal("Gagal menyimpan gambar produk")
	}
	// Best-effort: only clean up the old main image after the DB update
	// succeeds and the field actually changed.
	if existing.MainImageUrl != nil && entity.MainImageUrl != nil && *existing.MainImageUrl != *entity.MainImageUrl {
		_ = s.upload.DeleteFile(*existing.MainImageUrl)
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetPublished(ctx context.Context, id int64, published bool, actorID int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Produk tidak ditemukan")
	}
	if err := s.repo.SetPublished(ctx, id, published, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) SetFeatured(ctx context.Context, id int64, featured bool, actorID int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Produk tidak ditemukan")
	}
	if err := s.repo.SetFeatured(ctx, id, featured, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Produk tidak ditemukan")
	}
	images, err := s.repo.ListImages(ctx, id)
	if err != nil {
		return apperror.Internal("")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	// DB first, then files — a failed file delete never leaves a dangling DB row.
	if existing.MainImageUrl != nil {
		_ = s.upload.DeleteFile(*existing.MainImageUrl)
	}
	for _, img := range images {
		_ = s.upload.DeleteFile(img.ImageUrl)
	}
	return nil
}

func (s *ServiceImpl) PublicCategories(ctx context.Context) ([]goods_model.Category, error) {
	return s.categories(ctx, true)
}

func (s *ServiceImpl) CMSCategories(ctx context.Context) ([]goods_model.Category, error) {
	return s.categories(ctx, false)
}

func (s *ServiceImpl) categories(ctx context.Context, activeOnly bool) ([]goods_model.Category, error) {
	data, err := s.repo.CategoryList(ctx, activeOnly)
	if err != nil {
		return nil, apperror.Internal("")
	}
	if data == nil {
		data = []goods_model.Category{}
	}
	return data, nil
}

func (s *ServiceImpl) CategoryCreate(ctx context.Context, req goods_dto.CategoryRequest, actorID int64) (goods_model.Category, error) {
	slugStr, err := s.uniqueCategorySlug(ctx, req.CategoryName, 0)
	if err != nil {
		return goods_model.Category{}, err
	}
	cat := goods_model.Category{CategoryName: req.CategoryName, CategorySlug: slugStr, IsActive: req.IsActive, SortOrder: req.SortOrder}
	id, err := s.repo.CategoryCreate(ctx, cat, actorID)
	if err != nil {
		return goods_model.Category{}, apperror.Internal("Gagal menyimpan kategori")
	}
	return s.repo.CategoryFindByID(ctx, id)
}

func (s *ServiceImpl) CategoryUpdate(ctx context.Context, id int64, req goods_dto.CategoryRequest, actorID int64) (goods_model.Category, error) {
	existing, err := s.repo.CategoryFindByID(ctx, id)
	if err != nil {
		return goods_model.Category{}, apperror.NotFound("Kategori tidak ditemukan")
	}
	slugStr := existing.CategorySlug
	if existing.CategoryName != req.CategoryName {
		slugStr, err = s.uniqueCategorySlug(ctx, req.CategoryName, id)
		if err != nil {
			return goods_model.Category{}, err
		}
	}
	cat := goods_model.Category{CategoryName: req.CategoryName, CategorySlug: slugStr, IsActive: req.IsActive, SortOrder: req.SortOrder}
	if err := s.repo.CategoryUpdate(ctx, id, cat, actorID); err != nil {
		return goods_model.Category{}, apperror.Internal("")
	}
	return s.repo.CategoryFindByID(ctx, id)
}

func (s *ServiceImpl) CategoryDelete(ctx context.Context, id int64) error {
	if _, err := s.repo.CategoryFindByID(ctx, id); err != nil {
		return apperror.NotFound("Kategori tidak ditemukan")
	}
	inUse, err := s.repo.CategoryInUse(ctx, id)
	if err != nil {
		return apperror.Internal("")
	}
	if inUse {
		return apperror.Conflict("Kategori masih digunakan oleh produk dan tidak dapat dihapus")
	}
	if err := s.repo.CategoryDelete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) validateRequest(ctx context.Context, req goods_dto.Request, exceptID int64) error {
	if _, err := s.repo.CategoryFindByID(ctx, req.GoodsCategoryID); err != nil {
		return apperror.BadRequest("Kategori produk tidak ditemukan")
	}
	if req.SKUCode != "" {
		exists, err := s.repo.SKUExists(ctx, req.SKUCode, exceptID)
		if err != nil {
			return apperror.Internal("")
		}
		if exists {
			return apperror.BadRequest("SKU/kode produk sudah digunakan")
		}
	}
	if !isSafeURL(req.PurchaseUrl) {
		return apperror.BadRequest("Purchase URL harus berupa URL http/https yang valid")
	}
	return nil
}

// isSafeURL only allows absolute http/https URLs — rejects javascript:/data:
// and other schemes that would be unsafe to redirect the purchase button to.
func isSafeURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *ServiceImpl) fromRequest(req goods_dto.Request) goods_model.Goods {
	label := req.PurchaseButtonLabel
	if label == "" {
		label = "Beli Sekarang"
	}
	return goods_model.Goods{
		GoodsName:           req.GoodsName,
		SKUCode:             ptr.Str(req.SKUCode),
		GoodsCategoryID:     req.GoodsCategoryID,
		ShortDescription:    ptr.Str(req.ShortDescription),
		FullDescription:     ptr.Str(richTextPolicy.Sanitize(req.FullDescription)),
		Price:               req.Price,
		MainImageUrl:        ptr.Str(req.MainImageUrl),
		AvailabilityStatus:  req.AvailabilityStatus,
		PurchaseUrl:         req.PurchaseUrl,
		PurchaseButtonLabel: label,
	}
}

func (s *ServiceImpl) uniqueSlug(ctx context.Context, name string, exceptID int64) (string, error) {
	base := slug.Make(name)
	candidate := base
	for i := 2; i < 100; i++ {
		exists, err := s.repo.SlugExists(ctx, candidate, exceptID)
		if err != nil {
			return "", apperror.Internal("")
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, exceptID), nil
}

func (s *ServiceImpl) uniqueCategorySlug(ctx context.Context, name string, exceptID int64) (string, error) {
	base := slug.Make(name)
	candidate := base
	for i := 2; i < 100; i++ {
		exists, err := s.repo.CategorySlugExists(ctx, candidate, exceptID)
		if err != nil {
			return "", apperror.Internal("")
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, exceptID), nil
}
