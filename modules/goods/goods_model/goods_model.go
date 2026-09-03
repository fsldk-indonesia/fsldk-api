// Package goods_model memuat entitas modul goods. Murni struct data.
package goods_model

import "time"

// Goods merepresentasikan satu baris ms_goods.
type Goods struct {
	GoodsID             int64      `gorm:"column:goodsID;primaryKey" json:"goodsID"`
	GoodsName           string     `gorm:"column:goodsName" json:"goodsName"`
	GoodsSlug           string     `gorm:"column:goodsSlug" json:"goodsSlug"`
	SKUCode             *string    `gorm:"column:skuCode" json:"skuCode"`
	GoodsCategoryID     int64      `gorm:"column:goodsCategoryID" json:"goodsCategoryID"`
	CategoryName        string     `gorm:"column:categoryName;->" json:"categoryName"`
	ShortDescription    *string    `gorm:"column:shortDescription" json:"shortDescription"`
	FullDescription     *string    `gorm:"column:fullDescription" json:"fullDescription"`
	Price               float64    `gorm:"column:price" json:"price"`
	MainImageUrl        *string    `gorm:"column:mainImageUrl" json:"mainImageUrl"`
	AvailabilityStatus  string     `gorm:"column:availabilityStatus" json:"availabilityStatus"`
	IsFeatured          bool       `gorm:"column:isFeatured" json:"isFeatured"`
	IsPublished         bool       `gorm:"column:isPublished" json:"isPublished"`
	PublishedDate       *time.Time `gorm:"column:publishedDate" json:"publishedDate"`
	SortOrder           int        `gorm:"column:sortOrder" json:"sortOrder"`
	PurchaseUrl         string     `gorm:"column:purchaseUrl" json:"purchaseUrl"`
	PurchaseButtonLabel string     `gorm:"column:purchaseButtonLabel" json:"purchaseButtonLabel"`
	CreatedDate         time.Time  `gorm:"column:createdDate" json:"createdDate"`
}

// Image merepresentasikan satu baris ms_goods_image.
type Image struct {
	GoodsImageID int64  `gorm:"column:goodsImageID;primaryKey" json:"goodsImageID"`
	GoodsID      int64  `gorm:"column:goodsID" json:"goodsID"`
	ImageUrl     string `gorm:"column:imageUrl" json:"imageUrl"`
	SortOrder    int    `gorm:"column:sortOrder" json:"sortOrder"`
}

// Category merepresentasikan satu baris lk_goods_category.
type Category struct {
	GoodsCategoryID int64  `gorm:"column:goodsCategoryID;primaryKey" json:"goodsCategoryID"`
	CategoryName    string `gorm:"column:categoryName" json:"categoryName"`
	CategorySlug    string `gorm:"column:categorySlug" json:"categorySlug"`
	IsActive        bool   `gorm:"column:isActive" json:"isActive"`
	SortOrder       int    `gorm:"column:sortOrder" json:"sortOrder"`
}
