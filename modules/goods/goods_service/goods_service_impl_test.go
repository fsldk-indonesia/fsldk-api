package goods_service

import (
	"context"
	"errors"
	"testing"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/goods/goods_dto"
	"fsldk-api/modules/goods/goods_model"
)

// fakeGoodsRepository adalah implementasi goods_repository.Repository
// in-memory untuk menguji business logic goods_service tanpa DB sungguhan —
// hanya method yang benar-benar dipakai skenario uji di file ini yang punya
// perilaku bermakna, sisanya no-op.
type fakeGoodsRepository struct {
	goods         map[int64]goods_model.Goods
	categories    map[int64]goods_model.Category
	slugs         map[string]int64
	skus          map[string]int64
	images        map[int64][]string
	categoryInUse bool
	nextID        int64
	deleted       []int64
}

func newFakeGoodsRepository() *fakeGoodsRepository {
	return &fakeGoodsRepository{
		goods: map[int64]goods_model.Goods{}, categories: map[int64]goods_model.Category{},
		slugs: map[string]int64{}, skus: map[string]int64{}, images: map[int64][]string{},
	}
}

func (f *fakeGoodsRepository) List(ctx context.Context, filter goods_dto.Filter) ([]goods_model.Goods, int64, error) {
	return nil, 0, nil
}

func (f *fakeGoodsRepository) FindByID(ctx context.Context, id int64) (goods_model.Goods, error) {
	g, ok := f.goods[id]
	if !ok {
		return goods_model.Goods{}, errors.New("not found")
	}
	return g, nil
}

func (f *fakeGoodsRepository) FindBySlug(ctx context.Context, slug string) (goods_model.Goods, error) {
	return goods_model.Goods{}, errors.New("not found")
}

func (f *fakeGoodsRepository) SlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	id, ok := f.slugs[slug]
	if !ok {
		return false, nil
	}
	return id != exceptID, nil
}

func (f *fakeGoodsRepository) SKUExists(ctx context.Context, sku string, exceptID int64) (bool, error) {
	id, ok := f.skus[sku]
	if !ok {
		return false, nil
	}
	return id != exceptID, nil
}

func (f *fakeGoodsRepository) Create(ctx context.Context, g goods_model.Goods, createdBy int64) (int64, error) {
	f.nextID++
	id := f.nextID
	g.GoodsID = id
	f.goods[id] = g
	f.slugs[g.GoodsSlug] = id
	if g.SKUCode != nil && *g.SKUCode != "" {
		f.skus[*g.SKUCode] = id
	}
	return id, nil
}

func (f *fakeGoodsRepository) Update(ctx context.Context, id int64, g goods_model.Goods, updatedBy int64) error {
	g.GoodsID = id
	f.goods[id] = g
	f.slugs[g.GoodsSlug] = id
	return nil
}

func (f *fakeGoodsRepository) SetPublished(ctx context.Context, id int64, published bool, updatedBy int64) error {
	g := f.goods[id]
	g.IsPublished = published
	f.goods[id] = g
	return nil
}

func (f *fakeGoodsRepository) SetFeatured(ctx context.Context, id int64, featured bool, updatedBy int64) error {
	g := f.goods[id]
	g.IsFeatured = featured
	f.goods[id] = g
	return nil
}

func (f *fakeGoodsRepository) Delete(ctx context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	delete(f.goods, id)
	return nil
}

func (f *fakeGoodsRepository) ReplaceImages(ctx context.Context, goodsID int64, urls []string) error {
	f.images[goodsID] = urls
	return nil
}

func (f *fakeGoodsRepository) ListImages(ctx context.Context, goodsID int64) ([]goods_model.Image, error) {
	urls := f.images[goodsID]
	out := make([]goods_model.Image, len(urls))
	for i, u := range urls {
		out[i] = goods_model.Image{GoodsImageID: int64(i + 1), GoodsID: goodsID, ImageUrl: u, SortOrder: i}
	}
	return out, nil
}

func (f *fakeGoodsRepository) CategoryList(ctx context.Context, activeOnly bool) ([]goods_model.Category, error) {
	return nil, nil
}

func (f *fakeGoodsRepository) CategoryFindByID(ctx context.Context, id int64) (goods_model.Category, error) {
	c, ok := f.categories[id]
	if !ok {
		return goods_model.Category{}, errors.New("not found")
	}
	return c, nil
}

func (f *fakeGoodsRepository) CategorySlugExists(ctx context.Context, slug string, exceptID int64) (bool, error) {
	return false, nil
}

func (f *fakeGoodsRepository) CategoryCreate(ctx context.Context, cat goods_model.Category, createdBy int64) (int64, error) {
	f.nextID++
	id := f.nextID
	cat.GoodsCategoryID = id
	f.categories[id] = cat
	return id, nil
}

func (f *fakeGoodsRepository) CategoryUpdate(ctx context.Context, id int64, cat goods_model.Category, updatedBy int64) error {
	cat.GoodsCategoryID = id
	f.categories[id] = cat
	return nil
}

func (f *fakeGoodsRepository) CategoryDelete(ctx context.Context, id int64) error {
	delete(f.categories, id)
	return nil
}

func (f *fakeGoodsRepository) CategoryInUse(ctx context.Context, id int64) (bool, error) {
	return f.categoryInUse, nil
}

// fakeFileDeleter adalah implementasi FileDeleter in-memory — mencatat URL
// yang "dihapus" tanpa menyentuh disk, dipakai memverifikasi guard cleanup
// gambar saat Delete()/Update().
type fakeFileDeleter struct{ deletedURLs []string }

func (f *fakeFileDeleter) DeleteFile(publicURL string) error {
	f.deletedURLs = append(f.deletedURLs, publicURL)
	return nil
}

const validPurchaseURL = "https://wa.me/6281234567890"

func validRequest() goods_dto.Request {
	return goods_dto.Request{
		GoodsName: "Kaos FSLDK", GoodsCategoryID: 1, Price: 100000,
		AvailabilityStatus: constants.GoodsAvailable, PurchaseUrl: validPurchaseURL,
	}
}

func TestCreate_RejectsUnknownCategory(t *testing.T) {
	repo := newFakeGoodsRepository()
	svc := NewService(repo, &fakeFileDeleter{})

	_, err := svc.Create(context.Background(), validRequest(), 1)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected validation error for unknown category, got %v", err)
	}
}

func TestCreate_RejectsJavascriptSchemePurchaseURL(t *testing.T) {
	repo := newFakeGoodsRepository()
	repo.categories[1] = goods_model.Category{GoodsCategoryID: 1, CategoryName: "Merchandise", IsActive: true}
	svc := NewService(repo, &fakeFileDeleter{})

	req := validRequest()
	req.PurchaseUrl = "javascript:alert(1)"
	_, err := svc.Create(context.Background(), req, 1)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected validation error for javascript: purchase URL, got %v", err)
	}
}

func TestCreate_RejectsDuplicateSKU(t *testing.T) {
	repo := newFakeGoodsRepository()
	repo.categories[1] = goods_model.Category{GoodsCategoryID: 1, CategoryName: "Merchandise", IsActive: true}
	repo.skus["GDS-01"] = 99
	svc := NewService(repo, &fakeFileDeleter{})

	req := validRequest()
	req.SKUCode = "GDS-01"
	_, err := svc.Create(context.Background(), req, 1)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeValidationError {
		t.Fatalf("expected validation error for duplicate SKU, got %v", err)
	}
}

func TestCreate_GeneratesUniqueSlugOnCollision(t *testing.T) {
	repo := newFakeGoodsRepository()
	repo.categories[1] = goods_model.Category{GoodsCategoryID: 1, CategoryName: "Merchandise", IsActive: true}
	repo.slugs["kaos-fsldk"] = 999 // simulasikan produk lain sudah memakai slug dasar ini
	svc := NewService(repo, &fakeFileDeleter{})

	got, err := svc.Create(context.Background(), validRequest(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GoodsSlug != "kaos-fsldk-2" {
		t.Fatalf("expected slug suffixed to avoid collision, got %q", got.GoodsSlug)
	}
}

func TestDelete_CleansUpMainImageAndGallery(t *testing.T) {
	repo := newFakeGoodsRepository()
	mainURL := "https://api.example.com/uploads/main.jpg"
	repo.goods[1] = goods_model.Goods{GoodsID: 1, GoodsName: "Kaos", GoodsSlug: "kaos", MainImageUrl: &mainURL}
	repo.images[1] = []string{"https://api.example.com/uploads/g1.jpg", "https://api.example.com/uploads/g2.jpg"}
	uploader := &fakeFileDeleter{}
	svc := NewService(repo, uploader)

	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uploader.deletedURLs) != 3 {
		t.Fatalf("expected main image + 2 gallery images cleaned up, got %d: %v", len(uploader.deletedURLs), uploader.deletedURLs)
	}
	if _, ok := repo.goods[1]; ok {
		t.Fatalf("expected goods row to be removed from repository")
	}
}

func TestCategoryDelete_RejectsWhenInUse(t *testing.T) {
	repo := newFakeGoodsRepository()
	repo.categories[1] = goods_model.Category{GoodsCategoryID: 1, CategoryName: "Merchandise"}
	repo.categoryInUse = true
	svc := NewService(repo, &fakeFileDeleter{})

	err := svc.CategoryDelete(context.Background(), 1)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != constants.CodeConflict {
		t.Fatalf("expected conflict error when category still used by a product, got %v", err)
	}
}

func TestCategoryDelete_SucceedsWhenNotInUse(t *testing.T) {
	repo := newFakeGoodsRepository()
	repo.categories[1] = goods_model.Category{GoodsCategoryID: 1, CategoryName: "Merchandise"}
	svc := NewService(repo, &fakeFileDeleter{})

	if err := svc.CategoryDelete(context.Background(), 1); err != nil {
		t.Fatalf("expected delete to succeed when category is not in use, got error: %v", err)
	}
	if _, ok := repo.categories[1]; ok {
		t.Fatalf("expected category to be removed from repository")
	}
}
