package rapimnas_service

import (
	"context"
	"testing"

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
