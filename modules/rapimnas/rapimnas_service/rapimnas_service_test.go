package rapimnas_service

import (
	"context"
	"errors"
	"testing"

	"fsldk-api/modules/rapimnas/rapimnas_dto"
	"fsldk-api/modules/rapimnas/rapimnas_model"
)

// fakeRepo is a minimal in-memory stand-in for rapimnas_repository.Repository.
type fakeRepo struct {
	data       rapimnas_model.Aggregate
	saveErr    error
	saveCalled bool
}

func (f *fakeRepo) Get(ctx context.Context) (*rapimnas_model.Aggregate, error) {
	agg := f.data
	return &agg, nil
}

func (f *fakeRepo) Save(ctx context.Context, agg *rapimnas_model.Aggregate) error {
	f.saveCalled = true
	if f.saveErr != nil {
		return f.saveErr
	}
	f.data = *agg
	return nil
}

func TestGetPublic_FiltersHiddenResourcesAndNestsRundownEvents(t *testing.T) {
	repo := &fakeRepo{data: rapimnas_model.Aggregate{
		Setting: rapimnas_model.Setting{HeroTitle: "Rapimnas 2026"},
		Resources: []rapimnas_model.Resource{
			{Title: "Logo", IsVisible: true},
			{Title: "Twibbon", IsVisible: false},
		},
		RundownDays: []rapimnas_model.RundownDay{{ID: 1, DayLabel: "Hari 1"}},
		RundownEvents: []rapimnas_model.RundownEvent{
			{DayID: 1, Title: "Kedatangan Peserta"},
		},
	}}
	svc := NewService(repo)

	res, err := svc.GetPublic(context.Background())
	if err != nil {
		t.Fatalf("GetPublic() error = %v", err)
	}
	if len(res.Resources) != 1 || res.Resources[0].Title != "Logo" {
		t.Errorf("GetPublic().Resources = %+v, want exactly the 1 visible resource [Logo]", res.Resources)
	}
	if len(res.Rundown) != 1 || len(res.Rundown[0].Events) != 1 || res.Rundown[0].Events[0].Title != "Kedatangan Peserta" {
		t.Errorf("GetPublic().Rundown = %+v, want 1 day nesting 1 event (Kedatangan Peserta)", res.Rundown)
	}
}

func TestGet_CMS_IncludesHiddenResourcesAndAuditFields(t *testing.T) {
	actorID := int64(7)
	repo := &fakeRepo{data: rapimnas_model.Aggregate{
		Setting: rapimnas_model.Setting{HeroTitle: "Rapimnas 2026", UpdatedBy: &actorID},
		Resources: []rapimnas_model.Resource{
			{Title: "Logo", IsVisible: true},
			{Title: "Twibbon", IsVisible: false},
		},
	}}
	svc := NewService(repo)

	res, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(res.Resources) != 2 {
		t.Errorf("Get().Resources has %d items, want 2 (CMS sees hidden resources too)", len(res.Resources))
	}
	if res.UpdatedBy == nil || *res.UpdatedBy != actorID {
		t.Errorf("Get().UpdatedBy = %v, want %d", res.UpdatedBy, actorID)
	}
}

func TestJsonStrArray_NullJSONStringReturnsEmptySliceNotNil(t *testing.T) {
	nullStr := "null"
	got := jsonStrArray(&nullStr)
	if got == nil {
		t.Fatal("jsonStrArray(&\"null\") returned nil, want non-nil empty slice (so it marshals to JSON [] not null)")
	}
	if len(got) != 0 {
		t.Errorf("jsonStrArray(&\"null\") = %+v, want empty slice", got)
	}
}

func validUpdateRequest() rapimnas_dto.UpdateRequest {
	return rapimnas_dto.UpdateRequest{
		HeroBadgeText: "RAPIMNAS 1", HeroTitle: "Rapimnas 2026", HeroDateRangeText: "12-15 Nov 2026",
		HeroTaglineQuote: "Diponegoro's Spirit", CountdownTargetDate: "2026-11-12T08:00:00+07:00",
		Feature1IconKey: "user-group", Feature1Title: "Ukhuwah", Feature1Desc: "desc",
		Feature2IconKey: "file-sliders", Feature2Title: "Sistem", Feature2Desc: "desc",
		CtaTitle: "Mari Berkontribusi!", CtaDescription: "desc", CtaButtonLabel: "Daftar",
		FooterContactEmail: "rapimnasone@gmail.com", FooterCopyrightText: "(c) 2026",
		FooterIgHandle: "@rapimnas.fsldk", FooterIgUrl: "https://instagram.com/rapimnas.fsldk",
		FooterTiktokHandle: "@rapimnas.fsldk", FooterTiktokUrl: "https://tiktok.com/@rapimnas.fsldk",
		JadwalHeaderSubtitle: "subtitle",
		TentangTaglineQuote:  "quote", TentangDescParagraph1: "p1", TentangDescParagraph2: "p2", TentangVisiText: "visi",
		PesertaEarlyBirdDateRange: "6-12 Okt", PesertaRegulerDateRange: "13-26 Okt",
		PesertaBankName: "BSI", PesertaBankAccountNumber: "123", PesertaBankAccountHolder: "NAURA",
		PesertaGoogleFormUrl: "https://bit.ly/x",
		PanitiaClosedMessage: "belum dibuka",
	}
}

func TestUpdate_PersistsSubmittedOrder_NotAppendOnly(t *testing.T) {
	repo := &fakeRepo{data: rapimnas_model.Aggregate{
		GalleryImages: []rapimnas_model.GalleryImage{
			{ID: 1, ImageUrl: "old1.jpg"}, {ID: 2, ImageUrl: "old2.jpg"}, {ID: 3, ImageUrl: "old3.jpg"},
			{ID: 4, ImageUrl: "old4.jpg"}, {ID: 5, ImageUrl: "old5.jpg"},
		},
	}}
	svc := NewService(repo)

	req := validUpdateRequest()
	req.GalleryImages = []rapimnas_dto.GalleryImageDTO{
		{ImageUrl: "c.jpg"}, {ImageUrl: "a.jpg"}, {ImageUrl: "b.jpg"},
	}

	if _, err := svc.Update(context.Background(), req, 99); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got := repo.data.GalleryImages
	if len(got) != 3 {
		t.Fatalf("GalleryImages after Update() has %d items, want exactly 3 (replace, not append onto the old 5)", len(got))
	}
	if got[0].ImageUrl != "c.jpg" || got[1].ImageUrl != "a.jpg" || got[2].ImageUrl != "b.jpg" {
		t.Errorf("GalleryImages after Update() = %+v, want submitted order [c.jpg a.jpg b.jpg]", got)
	}
	if got[0].SortOrder != 0 || got[1].SortOrder != 1 || got[2].SortOrder != 2 {
		t.Errorf("GalleryImages SortOrder = [%d %d %d], want fresh [0 1 2] from array position", got[0].SortOrder, got[1].SortOrder, got[2].SortOrder)
	}
}

func TestUpdate_PropagatesRepositorySaveErrorWithoutReportingSuccess(t *testing.T) {
	repo := &fakeRepo{saveErr: errors.New("boom")}
	svc := NewService(repo)

	_, err := svc.Update(context.Background(), validUpdateRequest(), 1)
	if err == nil {
		t.Fatal("Update() error = nil, want a propagated error when repo.Save() fails")
	}
	if !repo.saveCalled {
		t.Error("repo.Save() was never called")
	}
}

func TestUpdate_RejectsInvalidCountdownDateWithoutCallingSave(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)

	req := validUpdateRequest()
	req.CountdownTargetDate = "not-a-date"

	_, err := svc.Update(context.Background(), req, 1)
	if err == nil {
		t.Fatal("Update() error = nil, want a validation error for an unparseable countdownTargetDate")
	}
	if repo.saveCalled {
		t.Error("repo.Save() was called despite an invalid countdownTargetDate — should fail before ever reaching the repository")
	}
}
