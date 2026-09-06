package dynamicform_service

import (
	"testing"

	"fsldk-api/modules/dynamicform/dynamicform_model"
)

func ptr(s string) *string { return &s }

// field is a tiny builder for test fixtures.
func field(id int64, typ string, cfg string) dynamicform_model.Field {
	f := dynamicform_model.Field{FieldID: id, FieldType: typ, IsActive: true}
	if cfg != "" {
		f.FieldConfigJSON = ptr(cfg)
	}
	return f
}

func TestBuildSections(t *testing.T) {
	fields := []dynamicform_model.Field{
		field(1, "email", ""),
		field(2, "short_text", ""),
		field(10, "section_break", ""),
		field(11, "radio", ""),
		field(20, "section_break", ""),
		field(21, "long_text", ""),
	}
	got := buildSections(fields)
	if len(got) != 3 {
		t.Fatalf("want 3 sections, got %d", len(got))
	}
	if got[0].BreakFieldID != 0 || len(got[0].FieldIDs) != 2 {
		t.Fatalf("section 0 = %+v", got[0])
	}
	if got[1].BreakFieldID != 10 || len(got[1].FieldIDs) != 1 {
		t.Fatalf("section 1 = %+v", got[1])
	}
	if got[2].BreakFieldID != 20 || got[2].FieldIDs[0] != 21 {
		t.Fatalf("section 2 = %+v", got[2])
	}
}

func TestReachableFieldIDs_ForwardJumpSkipsMiddleSection(t *testing.T) {
	// s0: email(1) + routing radio(2) -> "skip" jumps to section starting at break 20.
	// s1 (break 10): required long_text(11)  <- must be skippable
	// s2 (break 20): short_text(21)
	routing := `{"sectionRouting":{"enabled":true,"routes":[{"optionValue":"skip","targetSectionFieldID":20}]}}`
	fields := []dynamicform_model.Field{
		field(1, "email", ""),
		field(2, "radio", routing),
		field(10, "section_break", ""),
		field(11, "long_text", ""),
		field(20, "section_break", ""),
		field(21, "short_text", ""),
	}
	sections := buildSections(fields)

	// answered "skip" -> section 1 (field 11) unreachable, section 2 reachable.
	reach := reachableFieldIDs(fields, sections, map[int64][]string{2: {"skip"}})
	if !reach[1] || !reach[2] || !reach[21] {
		t.Fatalf("section 0 + target must be reachable: %v", reach)
	}
	if reach[11] {
		t.Fatalf("skipped section's field 11 must be unreachable: %v", reach)
	}

	// answered anything else -> fall through, every section reachable.
	reach2 := reachableFieldIDs(fields, sections, map[int64][]string{2: {"stay"}})
	if !reach2[11] || !reach2[21] {
		t.Fatalf("no matching route -> linear path, all reachable: %v", reach2)
	}
}

func TestReachableFieldIDs_BackwardTargetIgnored(t *testing.T) {
	// A route pointing at an earlier/same section must be ignored (forward-only).
	routing := `{"sectionRouting":{"enabled":true,"routes":[{"optionValue":"back","targetSectionFieldID":0}]}}`
	fields := []dynamicform_model.Field{
		field(10, "section_break", ""),
		field(11, "radio", routing),
		field(20, "section_break", ""),
		field(21, "short_text", ""),
	}
	sections := buildSections(fields)
	reach := reachableFieldIDs(fields, sections, map[int64][]string{11: {"back"}})
	if !reach[21] {
		t.Fatalf("forward-only: linear fall-through still reaches section 2: %v", reach)
	}
}

func TestReachableFieldIDs_NoBreaksAllReachable(t *testing.T) {
	fields := []dynamicform_model.Field{
		field(1, "email", ""),
		field(2, "long_text", ""),
	}
	reach := reachableFieldIDs(fields, buildSections(fields), map[int64][]string{})
	if !reach[1] || !reach[2] {
		t.Fatalf("single implicit section: all fields reachable: %v", reach)
	}
}
