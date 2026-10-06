package rapimnas_service

import (
	"context"
	"encoding/json"

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

// Update placeholder — replaced with the real implementation in Task 10.
func (s *ServiceImpl) Update(ctx context.Context, req rapimnas_dto.UpdateRequest, actorID int64) (rapimnas_dto.CMSResponse, error) {
	return rapimnas_dto.CMSResponse{}, apperror.Internal("not implemented")
}
