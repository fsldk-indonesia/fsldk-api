// Package goods_dto memuat DTO request/response modul goods. Murni struct data.
package goods_dto

import "fsldk-api/modules/goods/goods_model"

// DetailResponse adalah produk beserta gallery gambarnya — dipakai endpoint
// detail publik & CMS get. Endpoint list tidak menyertakan gallery (cukup
// mainImageUrl) supaya tidak query gambar untuk tiap baris listing.
type DetailResponse struct {
	goods_model.Goods
	Images []string `json:"images"`
}

// Request adalah body membuat/memperbarui produk. Dipakai untuk create
// maupun update — form CMS selalu mengirim seluruh state gallery saat ini,
// jadi ImageUrls cukup []string biasa (bukan pointer nil-vs-empty seperti
// campaign) karena tidak ada kebutuhan partial update di form ini.
type Request struct {
	GoodsName           string   `json:"goodsName" validate:"required,min=3,max=255"`
	SKUCode             string   `json:"skuCode" validate:"max=50"`
	GoodsCategoryID     int64    `json:"goodsCategoryID" validate:"required"`
	ShortDescription    string   `json:"shortDescription" validate:"max=500"`
	FullDescription     string   `json:"fullDescription"`
	Price               float64  `json:"price" validate:"gte=0"`
	MainImageUrl        string   `json:"mainImageUrl" validate:"max=500"`
	ImageUrls           []string `json:"imageUrls" validate:"max=10,dive,max=500"`
	AvailabilityStatus  string   `json:"availabilityStatus" validate:"required,oneof=available out_of_stock coming_soon"`
	PurchaseUrl         string   `json:"purchaseUrl" validate:"required,max=500"`
	PurchaseButtonLabel string   `json:"purchaseButtonLabel" validate:"max=100"`
}

// PublishRequest adalah body mengubah status publikasi produk.
type PublishRequest struct {
	IsPublished bool `json:"isPublished"`
}

// FeaturedRequest adalah body mengubah status unggulan produk.
type FeaturedRequest struct {
	IsFeatured bool `json:"isFeatured"`
}

// Filter menampung parameter penyaringan daftar produk (repository & service).
type Filter struct {
	Search        string
	CategorySlug  string
	CategoryID    int64
	Availability  string
	FeaturedOnly  bool
	PublishedOnly bool
	Limit         int
	Offset        int
	OrderBy       string
}

// CategoryRequest adalah body membuat/memperbarui kategori produk.
type CategoryRequest struct {
	CategoryName string `json:"categoryName" validate:"required,min=2,max=100"`
	IsActive     bool   `json:"isActive"`
	SortOrder    int    `json:"sortOrder"`
}
