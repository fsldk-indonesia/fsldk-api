package dynamicform_service

import (
	"encoding/json"
	"strings"

	"fsldk-api/modules/dynamicform/dynamicform_model"
)

// Sectioning & section routing (techspec Part 2, K1/K3/B1).
//
// A "section" is the run of fields between two `section_break` fields — there is
// no section table. Section 0 is implicit (the form's own title/description);
// every later section starts at a `section_break`.
//
// A radio/dropdown may carry a forward-only routing rule inside its fieldConfig:
//
//	{"sectionRouting":{"enabled":true,"routes":[
//	    {"optionValue":"Yes","targetSectionFieldID":42}]}}
//
// where targetSectionFieldID is the fieldID of the `section_break` that starts
// the destination section. On submit, the server walks the path the answers
// actually take and only enforces `required` for fields on that path — fields in
// skipped sections are treated as nullable and their answers dropped (B1).

type sectionRoute struct {
	OptionValue          string `json:"optionValue"`
	TargetSectionFieldID int64  `json:"targetSectionFieldID"`
}

type sectionRouting struct {
	Enabled bool           `json:"enabled"`
	Routes  []sectionRoute `json:"routes"`
}

type fieldConfigEnvelope struct {
	SectionRouting *sectionRouting `json:"sectionRouting"`
}

// formSection is one resolved section.
type formSection struct {
	// BreakFieldID is the fieldID of the section_break that starts this section,
	// or 0 for the implicit first section.
	BreakFieldID int64
	// FieldIDs are the input+display fields that belong to this section, in order
	// (the section_break itself is not included).
	FieldIDs []int64
}

// buildSections splits ordered active fields into sections on `section_break`.
func buildSections(fields []dynamicform_model.Field) []formSection {
	sections := []formSection{{BreakFieldID: 0}}
	for _, f := range fields {
		if !f.IsActive {
			continue
		}
		if f.FieldType == "section_break" {
			sections = append(sections, formSection{BreakFieldID: f.FieldID})
			continue
		}
		last := &sections[len(sections)-1]
		last.FieldIDs = append(last.FieldIDs, f.FieldID)
	}
	return sections
}

// parseSectionRouting reads an enabled routing rule from a field's fieldConfig.
func parseSectionRouting(fieldConfigJSON *string) (sectionRouting, bool) {
	if fieldConfigJSON == nil || strings.TrimSpace(*fieldConfigJSON) == "" {
		return sectionRouting{}, false
	}
	var env fieldConfigEnvelope
	if json.Unmarshal([]byte(*fieldConfigJSON), &env) != nil || env.SectionRouting == nil {
		return sectionRouting{}, false
	}
	if !env.SectionRouting.Enabled || len(env.SectionRouting.Routes) == 0 {
		return sectionRouting{}, false
	}
	return *env.SectionRouting, true
}

// reachableFieldIDs walks the section path implied by `values` and returns the
// set of fieldIDs on that path. Routing may only jump forward.
func reachableFieldIDs(
	fields []dynamicform_model.Field,
	sections []formSection,
	values map[int64][]string,
) map[int64]bool {
	fieldsByID := make(map[int64]dynamicform_model.Field, len(fields))
	for _, f := range fields {
		fieldsByID[f.FieldID] = f
	}
	indexByBreak := make(map[int64]int, len(sections))
	for i, sec := range sections {
		indexByBreak[sec.BreakFieldID] = i
	}

	reachable := map[int64]bool{}
	for cur := 0; cur < len(sections); {
		sec := sections[cur]
		for _, id := range sec.FieldIDs {
			reachable[id] = true
		}

		next := cur + 1 // default: fall through to the next section
		for _, id := range sec.FieldIDs {
			f := fieldsByID[id]
			if f.FieldType != "radio" && f.FieldType != "dropdown" {
				continue
			}
			routing, ok := parseSectionRouting(f.FieldConfigJSON)
			if !ok {
				continue
			}
			answer := firstVal(values, id)
			if answer == "" {
				continue
			}
			for _, r := range routing.Routes {
				if r.OptionValue != answer {
					continue
				}
				if target, exists := indexByBreak[r.TargetSectionFieldID]; exists && target > cur {
					next = target
				}
				break
			}
			if next != cur+1 {
				break // first answered routing field with a forward target wins
			}
		}
		cur = next
	}
	return reachable
}
