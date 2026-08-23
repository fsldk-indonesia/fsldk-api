package shortlink_service

import (
	"context"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/security"
	"fsldk-api/modules/shortlink/shortlink_dto"
	"fsldk-api/modules/shortlink/shortlink_model"
	"fsldk-api/modules/shortlink/shortlink_repository"
)

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"shortKey":    "s.shortKey",
	"visitCount":  "s.visitCount",
	"createdDate": "s.createdDate",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo        shortlink_repository.Repository
	frontendURL string
}

// NewService membuat Service shortlink. frontendURL adalah base URL publik
// fsldk-web (mis. https://fsldk-indonesia.com) yang dipakai membentuk
// shortURL utuh — pengunjung membuka shortlink di domain frontend, yang lalu
// me-resolve & redirect ke destinationURL lewat endpoint publik backend ini.
func NewService(repo shortlink_repository.Repository, frontendURL string) Service {
	return &ServiceImpl{repo: repo, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func (s *ServiceImpl) toResponse(sl shortlink_model.ShortLink) shortlink_dto.Response {
	return shortlink_dto.Response{
		ShortLinkID:    sl.ShortLinkID,
		ShortKey:       sl.ShortKey,
		DestinationURL: sl.DestinationURL,
		ShortURL:       s.frontendURL + "/" + sl.ShortKey,
		VisitCount:     sl.VisitCount,
		AuthorName:     sl.AuthorName,
		CreatedDate:    sl.CreatedDate.Format("2006-01-02 15:04:05"),
	}
}

func (s *ServiceImpl) List(ctx context.Context, q dto.ListQuery) ([]shortlink_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, shortlink_dto.ListFilter{
		Search:  q.Search,
		Limit:   q.Limit,
		Offset:  q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "s.createdDate DESC"),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]shortlink_dto.Response, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toResponse(r))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (shortlink_dto.Response, error) {
	sl, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return shortlink_dto.Response{}, apperror.NotFound("Shortlink tidak ditemukan")
	}
	return s.toResponse(sl), nil
}

func (s *ServiceImpl) Create(ctx context.Context, req shortlink_dto.CreateRequest, actorID int64) (shortlink_dto.Response, error) {
	key := strings.TrimSpace(req.ShortKey)
	if key != "" {
		exists, err := s.repo.ExistsByKey(ctx, key, 0)
		if err != nil {
			return shortlink_dto.Response{}, apperror.Internal("")
		}
		if exists {
			return shortlink_dto.Response{}, apperror.Conflict("Kunci shortlink sudah dipakai")
		}
	} else {
		generated, err := s.generateUniqueKey(ctx)
		if err != nil {
			return shortlink_dto.Response{}, apperror.Internal("")
		}
		key = generated
	}

	id, err := s.repo.Create(ctx, key, req.DestinationURL, actorID)
	if err != nil {
		return shortlink_dto.Response{}, apperror.Internal("Gagal membuat shortlink")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req shortlink_dto.UpdateRequest, actorID int64) (shortlink_dto.Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return shortlink_dto.Response{}, apperror.NotFound("Shortlink tidak ditemukan")
	}
	key := strings.TrimSpace(req.ShortKey)
	exists, err := s.repo.ExistsByKey(ctx, key, id)
	if err != nil {
		return shortlink_dto.Response{}, apperror.Internal("")
	}
	if exists {
		return shortlink_dto.Response{}, apperror.Conflict("Kunci shortlink sudah dipakai")
	}
	if err := s.repo.Update(ctx, id, key, req.DestinationURL, actorID); err != nil {
		return shortlink_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Shortlink tidak ditemukan")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Resolve(ctx context.Context, key string) (string, error) {
	sl, err := s.repo.FindByKey(ctx, strings.TrimSpace(key))
	if err != nil {
		return "", apperror.NotFound("Tautan tidak ditemukan")
	}
	_ = s.repo.IncrementVisit(ctx, sl.ShortLinkID)
	return sl.DestinationURL, nil
}

// GenerateUniqueKey membuat kunci acak (8 karakter heksadesimal) yang belum
// dipakai — exported agar bisa dipakai ulang oleh shortlinkrequest_service.
func (s *ServiceImpl) GenerateUniqueKey(ctx context.Context) (string, error) {
	return s.generateUniqueKey(ctx)
}

// KeyExists mengembalikan true bila shortKey sudah dipakai shortlink manapun.
func (s *ServiceImpl) KeyExists(ctx context.Context, key string) (bool, error) {
	return s.repo.ExistsByKey(ctx, key, 0)
}

// generateUniqueKey membuat kunci acak (8 karakter heksadesimal) dan
// memastikan belum dipakai, dengan beberapa kali percobaan ulang.
func (s *ServiceImpl) generateUniqueKey(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		key, err := security.RandomToken(4)
		if err != nil {
			return "", err
		}
		exists, err := s.repo.ExistsByKey(ctx, key, 0)
		if err != nil {
			return "", err
		}
		if !exists {
			return key, nil
		}
	}
	return "", apperror.Internal("Gagal membuat kunci shortlink unik")
}
