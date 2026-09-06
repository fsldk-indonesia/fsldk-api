package contact_repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"fsldk-api/modules/contact/contact_dto"
	"fsldk-api/modules/contact/contact_model"

	"gorm.io/gorm"
)

type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new instance of contact Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) Create(ctx context.Context, msg *contact_model.ContactMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *repositoryImpl) FindByID(ctx context.Context, id int64) (*contact_model.ContactMessage, error) {
	var msg contact_model.ContactMessage
	err := r.db.WithContext(ctx).First(&msg, "messageID = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *repositoryImpl) FindAll(ctx context.Context, q contact_dto.ContactListQuery) ([]contact_model.ContactMessage, int64, error) {
	var messages []contact_model.ContactMessage
	var total int64

	db := r.db.WithContext(ctx).Model(&contact_model.ContactMessage{})

	if q.Search != "" {
		term := "%" + strings.TrimSpace(q.Search) + "%"
		db = db.Where("senderName LIKE ? OR email LIKE ? OR subject LIKE ?", term, term, term)
	}
	if q.IsRead != nil {
		db = db.Where("isRead = ?", *q.IsRead)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderCol := "createdDate"
	orderDir := "DESC"
	switch strings.ToLower(q.SortBy) {
	case "sendername":
		orderCol = "senderName"
	case "email":
		orderCol = "email"
	case "subject":
		orderCol = "subject"
	case "createddate":
		orderCol = "createdDate"
	}
	if strings.ToLower(q.SortOrder) == "asc" {
		orderDir = "ASC"
	}
	db = db.Order(fmt.Sprintf("%s %s", orderCol, orderDir))

	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 100 {
		limit = 15
	}
	offset := (page - 1) * limit

	if err := db.Limit(limit).Offset(offset).Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

func (r *repositoryImpl) MarkAsRead(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Model(&contact_model.ContactMessage{}).Where("messageID = ?", id).Update("isRead", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repositoryImpl) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&contact_model.ContactMessage{}, "messageID = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repositoryImpl) CountUnread(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&contact_model.ContactMessage{}).Where("isRead = ?", false).Count(&count).Error
	return int(count), err
}
