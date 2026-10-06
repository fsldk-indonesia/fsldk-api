package rapimnas_service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/rapimnas/rapimnas_dto"
	"fsldk-api/modules/rapimnas/rapimnas_model"
	"fsldk-api/modules/rapimnas/rapimnas_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo rapimnas_repository.Repository
}

// NewService membuat Service rapimnas.
func NewService(repo rapimnas_repository.Repository) Service {
	return &ServiceImpl{repo: repo}
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func jsonStrArray(raw *string) []string {
	if raw == nil || *raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(*raw), &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func toResourceDTO(r rapimnas_model.Resource) rapimnas_dto.ResourceDTO {
	return rapimnas_dto.ResourceDTO{
		Title: r.Title, Description: r.Description, IconKey: r.IconKey, Url: r.Url,
		ButtonLabel: r.ButtonLabel, IsVisible: r.IsVisible, SortOrder: r.SortOrder,
	}
}

// toPublicResponse membangun PublicResponse dari Aggregate — Resources
// HANYA yang IsVisible=true (lihat "Design decisions" item 3 di kepala plan
// ini), dan RundownEvents dinested ke bawah RundownDays masing-masing lewat
// pencocokan DayID == RundownDay.ID.
func toPublicResponse(agg rapimnas_model.Aggregate) rapimnas_dto.PublicResponse {
	s := agg.Setting

	countdown := ""
	if s.CountdownTargetDate != nil {
		countdown = s.CountdownTargetDate.Format("2006-01-02T15:04:05-07:00")
	}

	galleryImages := make([]rapimnas_dto.GalleryImageDTO, len(agg.GalleryImages))
	for i, g := range agg.GalleryImages {
		galleryImages[i] = rapimnas_dto.GalleryImageDTO{ImageUrl: g.ImageUrl, SortOrder: g.SortOrder}
	}

	homeCards := make([]rapimnas_dto.HomeCardDTO, len(agg.HomeCards))
	for i, c := range agg.HomeCards {
		homeCards[i] = rapimnas_dto.HomeCardDTO{IconKey: c.IconKey, Title: c.Title, Description: c.Description, SortOrder: c.SortOrder}
	}

	eventsByDay := make(map[int64][]rapimnas_dto.RundownEventDTO)
	for _, e := range agg.RundownEvents {
		eventsByDay[e.DayID] = append(eventsByDay[e.DayID], rapimnas_dto.RundownEventDTO{
			Time: e.Time, Title: e.Title, Description: e.Description, Venue: e.Venue, SortOrder: e.SortOrder,
		})
	}
	rundown := make([]rapimnas_dto.RundownDayDTO, len(agg.RundownDays))
	for i, d := range agg.RundownDays {
		rundown[i] = rapimnas_dto.RundownDayDTO{
			DayLabel: d.DayLabel, DateText: d.DateText, SortOrder: d.SortOrder,
			Events: eventsByDay[d.ID],
		}
	}

	visibleResources := make([]rapimnas_dto.ResourceDTO, 0, len(agg.Resources))
	for _, r := range agg.Resources {
		if !r.IsVisible {
			continue
		}
		visibleResources = append(visibleResources, toResourceDTO(r))
	}

	pickupLocations := make([]rapimnas_dto.PickupLocationDTO, len(agg.PickupLocations))
	for i, p := range agg.PickupLocations {
		pickupLocations[i] = rapimnas_dto.PickupLocationDTO{
			Name: p.Name, Type: p.Type, Description: p.Description, MapLink: p.MapLink, SortOrder: p.SortOrder,
		}
	}

	contacts := make([]rapimnas_dto.ContactDTO, len(agg.Contacts))
	for i, c := range agg.Contacts {
		contacts[i] = rapimnas_dto.ContactDTO{ContactType: c.ContactType, Name: c.Name, PhoneNumber: c.PhoneNumber, SortOrder: c.SortOrder}
	}

	return rapimnas_dto.PublicResponse{
		HeroBadgeText: s.HeroBadgeText, HeroTitle: s.HeroTitle, HeroDateRangeText: s.HeroDateRangeText,
		HeroTaglineQuote: s.HeroTaglineQuote, HeroImageUrl: strOrEmpty(s.HeroImageUrl), CountdownTargetDate: countdown,

		Feature1IconKey: s.Feature1IconKey, Feature1Title: s.Feature1Title, Feature1Desc: s.Feature1Desc,
		Feature2IconKey: s.Feature2IconKey, Feature2Title: s.Feature2Title, Feature2Desc: s.Feature2Desc,

		CtaTitle: s.CtaTitle, CtaDescription: s.CtaDescription, CtaButtonLabel: s.CtaButtonLabel,
		CtaMascotImageUrl: strOrEmpty(s.CtaMascotImageUrl),

		FooterContactEmail: s.FooterContactEmail, FooterCopyrightText: s.FooterCopyrightText,
		FooterIgHandle: s.FooterIgHandle, FooterIgUrl: s.FooterIgUrl,
		FooterTiktokHandle: s.FooterTiktokHandle, FooterTiktokUrl: s.FooterTiktokUrl,

		JadwalHeaderSubtitle: s.JadwalHeaderSubtitle,

		TentangTaglineQuote: s.TentangTaglineQuote, TentangDescParagraph1: s.TentangDescParagraph1,
		TentangDescParagraph2: s.TentangDescParagraph2, TentangVisiText: s.TentangVisiText,
		TentangMisi: jsonStrArray(s.TentangMisiJSON), TentangTujuan: jsonStrArray(s.TentangTujuanJSON),
		TentangKegiatan: jsonStrArray(s.TentangKegiatanJSON),

		PesertaEarlyBirdDateRange: s.PesertaEarlyBirdDateRange, PesertaRegulerDateRange: s.PesertaRegulerDateRange,
		PesertaHargaNonSemarangEarlyBird: s.PesertaHargaNonSemarangEarlyBird,
		PesertaHargaNonSemarangReguler:   s.PesertaHargaNonSemarangReguler,
		PesertaHargaSemarangEarlyBird:    s.PesertaHargaSemarangEarlyBird,
		PesertaHargaSemarangReguler:      s.PesertaHargaSemarangReguler,
		PesertaBankName:                  s.PesertaBankName, PesertaBankAccountNumber: s.PesertaBankAccountNumber,
		PesertaBankAccountHolder: s.PesertaBankAccountHolder,
		PesertaGuidebookUrl:      strOrEmpty(s.PesertaGuidebookUrl), PesertaGoogleFormUrl: s.PesertaGoogleFormUrl,
		PesertaMapEmbedUrl: strOrEmpty(s.PesertaMapEmbedUrl),

		PanitiaIsOpen: s.PanitiaIsOpen, PanitiaClosedMessage: s.PanitiaClosedMessage,

		GalleryImages: galleryImages, HomeCards: homeCards, Rundown: rundown,
		Resources: visibleResources, PickupLocations: pickupLocations, Contacts: contacts,
	}
}

// toCMSResponse sama seperti toPublicResponse tapi menimpa Resources dengan
// SELURUH baris (termasuk IsVisible=false) dan menambahkan metadata audit.
func toCMSResponse(agg rapimnas_model.Aggregate) rapimnas_dto.CMSResponse {
	pub := toPublicResponse(agg)

	allResources := make([]rapimnas_dto.ResourceDTO, len(agg.Resources))
	for i, r := range agg.Resources {
		allResources[i] = toResourceDTO(r)
	}
	pub.Resources = allResources

	updatedDate := ""
	if agg.Setting.UpdatedDate != nil {
		updatedDate = agg.Setting.UpdatedDate.Format("2006-01-02 15:04:05")
	}
	return rapimnas_dto.CMSResponse{
		PublicResponse: pub,
		UpdatedDate:    updatedDate,
		UpdatedBy:      agg.Setting.UpdatedBy,
	}
}

func (s *ServiceImpl) Get(ctx context.Context) (rapimnas_dto.CMSResponse, error) {
	agg, err := s.repo.Get(ctx)
	if err != nil {
		return rapimnas_dto.CMSResponse{}, apperror.Internal("")
	}
	return toCMSResponse(*agg), nil
}

func (s *ServiceImpl) GetPublic(ctx context.Context) (rapimnas_dto.PublicResponse, error) {
	agg, err := s.repo.Get(ctx)
	if err != nil {
		return rapimnas_dto.PublicResponse{}, apperror.Internal("")
	}
	return toPublicResponse(*agg), nil
}

// parseDateTime menerima beberapa format umum ISO8601/datetime-local; sama
// seperti helper sejenis di event_service_impl.go (duplikasi kecil ini
// konsisten dengan modul lain di codebase — setiap service punya copy-nya
// sendiri, bukan helper bersama).
func parseDateTime(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("cannot parse datetime: %q", s)
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// toAggregate meratakan UpdateRequest menjadi Aggregate. SortOrder tiap item
// diabaikan dan ditimpa dengan posisi array (fresh sortOrder, lihat Design
// decisions #2) — CountdownTargetDate SENGAJA tidak diisi di sini; Update()
// mem-parse & menimpanya setelah toAggregate() dipanggil, supaya kegagalan
// parse bisa ditolak sebelum membangun apa pun yang lain.
func toAggregate(req rapimnas_dto.UpdateRequest, actorID int64) rapimnas_model.Aggregate {
	misiJSON, _ := json.Marshal(req.TentangMisi)
	tujuanJSON, _ := json.Marshal(req.TentangTujuan)
	kegiatanJSON, _ := json.Marshal(req.TentangKegiatan)
	misiStr, tujuanStr, kegiatanStr := string(misiJSON), string(tujuanJSON), string(kegiatanJSON)

	setting := rapimnas_model.Setting{
		ID:                rapimnas_model.SingletonID,
		HeroBadgeText:     req.HeroBadgeText,
		HeroTitle:         req.HeroTitle,
		HeroDateRangeText: req.HeroDateRangeText,
		HeroTaglineQuote:  req.HeroTaglineQuote,
		HeroImageUrl:      nullableStr(req.HeroImageUrl),

		Feature1IconKey: req.Feature1IconKey, Feature1Title: req.Feature1Title, Feature1Desc: req.Feature1Desc,
		Feature2IconKey: req.Feature2IconKey, Feature2Title: req.Feature2Title, Feature2Desc: req.Feature2Desc,

		CtaTitle: req.CtaTitle, CtaDescription: req.CtaDescription, CtaButtonLabel: req.CtaButtonLabel,
		CtaMascotImageUrl: nullableStr(req.CtaMascotImageUrl),

		FooterContactEmail: req.FooterContactEmail, FooterCopyrightText: req.FooterCopyrightText,
		FooterIgHandle: req.FooterIgHandle, FooterIgUrl: req.FooterIgUrl,
		FooterTiktokHandle: req.FooterTiktokHandle, FooterTiktokUrl: req.FooterTiktokUrl,

		JadwalHeaderSubtitle: req.JadwalHeaderSubtitle,

		TentangTaglineQuote: req.TentangTaglineQuote, TentangDescParagraph1: req.TentangDescParagraph1,
		TentangDescParagraph2: req.TentangDescParagraph2, TentangVisiText: req.TentangVisiText,
		TentangMisiJSON: &misiStr, TentangTujuanJSON: &tujuanStr, TentangKegiatanJSON: &kegiatanStr,

		PesertaEarlyBirdDateRange: req.PesertaEarlyBirdDateRange, PesertaRegulerDateRange: req.PesertaRegulerDateRange,
		PesertaHargaNonSemarangEarlyBird: req.PesertaHargaNonSemarangEarlyBird,
		PesertaHargaNonSemarangReguler:   req.PesertaHargaNonSemarangReguler,
		PesertaHargaSemarangEarlyBird:    req.PesertaHargaSemarangEarlyBird,
		PesertaHargaSemarangReguler:      req.PesertaHargaSemarangReguler,
		PesertaBankName:                  req.PesertaBankName, PesertaBankAccountNumber: req.PesertaBankAccountNumber,
		PesertaBankAccountHolder: req.PesertaBankAccountHolder,
		PesertaGuidebookUrl:      nullableStr(req.PesertaGuidebookUrl), PesertaGoogleFormUrl: req.PesertaGoogleFormUrl,
		PesertaMapEmbedUrl: nullableStr(req.PesertaMapEmbedUrl),

		PanitiaIsOpen: req.PanitiaIsOpen, PanitiaClosedMessage: req.PanitiaClosedMessage,

		UpdatedBy: &actorID,
	}

	galleryImages := make([]rapimnas_model.GalleryImage, len(req.GalleryImages))
	for i, g := range req.GalleryImages {
		galleryImages[i] = rapimnas_model.GalleryImage{ImageUrl: g.ImageUrl, SortOrder: i}
	}

	homeCards := make([]rapimnas_model.HomeCard, len(req.HomeCards))
	for i, c := range req.HomeCards {
		homeCards[i] = rapimnas_model.HomeCard{IconKey: c.IconKey, Title: c.Title, Description: c.Description, SortOrder: i}
	}

	var rundownDays []rapimnas_model.RundownDay
	var rundownEvents []rapimnas_model.RundownEvent
	for dayIdx, d := range req.Rundown {
		correlationID := int64(dayIdx)
		rundownDays = append(rundownDays, rapimnas_model.RundownDay{
			ID: correlationID, DayLabel: d.DayLabel, DateText: d.DateText, SortOrder: dayIdx,
		})
		for evtIdx, e := range d.Events {
			rundownEvents = append(rundownEvents, rapimnas_model.RundownEvent{
				DayID: correlationID, Time: e.Time, Title: e.Title, Description: e.Description,
				Venue: e.Venue, SortOrder: evtIdx,
			})
		}
	}

	resources := make([]rapimnas_model.Resource, len(req.Resources))
	for i, r := range req.Resources {
		resources[i] = rapimnas_model.Resource{
			Title: r.Title, Description: r.Description, IconKey: r.IconKey, Url: r.Url,
			ButtonLabel: r.ButtonLabel, IsVisible: r.IsVisible, SortOrder: i,
		}
	}

	pickupLocations := make([]rapimnas_model.PickupLocation, len(req.PickupLocations))
	for i, p := range req.PickupLocations {
		pickupLocations[i] = rapimnas_model.PickupLocation{
			Name: p.Name, Type: p.Type, Description: p.Description, MapLink: p.MapLink, SortOrder: i,
		}
	}

	contacts := make([]rapimnas_model.Contact, len(req.Contacts))
	for i, c := range req.Contacts {
		contacts[i] = rapimnas_model.Contact{ContactType: c.ContactType, Name: c.Name, PhoneNumber: c.PhoneNumber, SortOrder: i}
	}

	return rapimnas_model.Aggregate{
		Setting: setting, GalleryImages: galleryImages, HomeCards: homeCards,
		RundownDays: rundownDays, RundownEvents: rundownEvents, Resources: resources,
		PickupLocations: pickupLocations, Contacts: contacts,
	}
}

func (s *ServiceImpl) Update(ctx context.Context, req rapimnas_dto.UpdateRequest, actorID int64) (rapimnas_dto.CMSResponse, error) {
	countdown, err := parseDateTime(req.CountdownTargetDate)
	if err != nil {
		return rapimnas_dto.CMSResponse{}, apperror.BadRequest("countdownTargetDate tidak valid")
	}

	now := time.Now()
	agg := toAggregate(req, actorID)
	agg.Setting.CountdownTargetDate = countdown
	agg.Setting.UpdatedDate = &now

	if err := s.repo.Save(ctx, &agg); err != nil {
		return rapimnas_dto.CMSResponse{}, apperror.Internal("")
	}
	return s.Get(ctx)
}
