package welcomepopup_service

import (
	"context"
	"encoding/json"
	"testing"

	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
	"fsldk-api/modules/welcomepopup/welcomepopup_model"
)

// fakeRepo is a minimal in-memory stand-in for welcomepopup_repository.Repository.
type fakeRepo struct {
	data welcomepopup_model.WelcomePopup
}

func (f *fakeRepo) Get(ctx context.Context) (welcomepopup_model.WelcomePopup, error) {
	return f.data, nil
}

func (f *fakeRepo) Update(ctx context.Context, popup welcomepopup_model.WelcomePopup) error {
	f.data = popup
	return nil
}

func strPtr(s string) *string { return &s }

func TestGetPublic_NeverExposesAuditFields(t *testing.T) {
	repo := &fakeRepo{data: welcomepopup_model.WelcomePopup{
		ID: welcomepopup_model.SingletonID, IsEnabled: true,
		HTMLContent: strPtr("<div>hi</div>"), JSContent: strPtr("alert(1)"), CSSContent: strPtr("div{color:red}"),
	}}
	svc := NewService(repo)

	res, err := svc.GetPublic(context.Background())
	if err != nil {
		t.Fatalf("GetPublic() error = %v", err)
	}

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var keys map[string]interface{}
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	want := map[string]bool{"isEnabled": true, "htmlContent": true, "jsContent": true, "cssContent": true}
	if len(keys) != len(want) {
		t.Fatalf("PublicResponse JSON has %d keys %v, want exactly %v", len(keys), keys, want)
	}
	for k := range keys {
		if !want[k] {
			t.Errorf("PublicResponse leaked unexpected field %q", k)
		}
	}
}

func TestUpdate_PersistsNewContentAndIsReflectedByGet(t *testing.T) {
	repo := &fakeRepo{data: welcomepopup_model.WelcomePopup{ID: welcomepopup_model.SingletonID, IsEnabled: false}}
	svc := NewService(repo)

	if _, err := svc.Update(context.Background(), welcomepopup_dto.UpdateRequest{
		IsEnabled: true, HTMLContent: "<div>new</div>", JSContent: "run();", CSSContent: "body{}",
	}, 42); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !got.IsEnabled || got.HTMLContent != "<div>new</div>" || got.JSContent != "run();" || got.CSSContent != "body{}" {
		t.Errorf("Get() after Update() = %+v, want the updated content", got)
	}
}

func TestGetPublic_DisabledStateIsNotFilteredByBackend(t *testing.T) {
	repo := &fakeRepo{data: welcomepopup_model.WelcomePopup{
		ID: welcomepopup_model.SingletonID, IsEnabled: false, HTMLContent: strPtr("<div>hi</div>"),
	}}
	svc := NewService(repo)

	res, err := svc.GetPublic(context.Background())
	if err != nil {
		t.Fatalf("GetPublic() error = %v", err)
	}
	if res.IsEnabled {
		t.Errorf("GetPublic().IsEnabled = true, want false — the backend must return the true state; filtering happens on the frontend")
	}
	if res.HTMLContent == "" {
		t.Errorf("GetPublic().HTMLContent should still be populated even when disabled")
	}
}
