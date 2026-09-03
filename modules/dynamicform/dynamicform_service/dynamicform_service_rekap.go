package dynamicform_service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/constants"
	"fsldk-api/modules/dynamicform/dynamicform_dto"
	"fsldk-api/modules/dynamicform/dynamicform_model"
	"fsldk-api/modules/dynamicform/dynamicform_repository"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
)

// dataFields returns the active non-display fields in sortOrder — the columns
// shared by the rekap table, the CSV export and the Google Sheet.
func dataFields(fields []dynamicform_model.Field) []dynamicform_model.Field {
	out := make([]dynamicform_model.Field, 0, len(fields))
	for _, f := range fields {
		if !isDisplayType(f.FieldType) {
			out = append(out, f)
		}
	}
	return out
}

// buildHeader is the single header builder shared by CSV and Sheets.
func buildHeader(fields []dynamicform_model.Field) []string {
	head := []string{"Timestamp", "Submission ID", "Email", "Nama"}
	for _, f := range dataFields(fields) {
		head = append(head, f.Label)
	}
	return head
}

// buildRow is the single row builder shared by CSV and Sheets.
func buildRow(sub dynamicform_model.Submission, answers map[int64]string, fields []dynamicform_model.Field) []string {
	row := []string{
		sub.SubmittedDate.Format(timeLayout),
		strconv.FormatInt(sub.SubmissionID, 10),
		sub.RespondentEmail,
		strDeref(sub.RespondentName),
	}
	for _, f := range dataFields(fields) {
		row = append(row, displayValue(answers[f.FieldID]))
	}
	return row
}

// ---------------------------------------------------------------------------
// rekap
// ---------------------------------------------------------------------------

func (s *ServiceImpl) ListSubmissions(ctx context.Context, formID int64, q dto.ListQuery, f dynamicform_dto.SubmissionFilter, actorID int64, perms []string) ([]dynamicform_dto.SubmissionRow, int, error) {
	if _, err := s.getOwnedForm(ctx, formID, actorID, perms); err != nil {
		return nil, 0, err
	}
	f.Limit = q.Limit
	f.Offset = q.Offset()
	if f.Search == "" {
		f.Search = q.Search
	}
	subs, total, err := s.repo.ListSubmissions(ctx, formID, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	ids := make([]int64, len(subs))
	for i, sub := range subs {
		ids[i] = sub.SubmissionID
	}
	answers, _ := s.repo.AnswersFor(ctx, ids)
	out := make([]dynamicform_dto.SubmissionRow, 0, len(subs))
	for _, sub := range subs {
		row := dynamicform_dto.SubmissionRow{
			SubmissionID: sub.SubmissionID, RespondentEmail: sub.RespondentEmail,
			RespondentName: strDeref(sub.RespondentName), IsValid: sub.IsValid,
			SubmittedDate: sub.SubmittedDate.Format(timeLayout), Answers: map[string]string{},
		}
		for fieldID, v := range answers[sub.SubmissionID] {
			row.Answers["field_"+strconv.FormatInt(fieldID, 10)] = displayValue(v)
		}
		out = append(out, row)
	}
	return out, int(total), nil
}

func (s *ServiceImpl) GetSubmission(ctx context.Context, formID, submissionID int64, actorID int64, perms []string) (dynamicform_dto.SubmissionDetail, error) {
	if _, err := s.getOwnedForm(ctx, formID, actorID, perms); err != nil {
		return dynamicform_dto.SubmissionDetail{}, err
	}
	sub, err := s.repo.GetSubmission(ctx, formID, submissionID)
	if err != nil {
		return dynamicform_dto.SubmissionDetail{}, apperror.NotFound("Tanggapan tidak ditemukan")
	}
	fields, _ := s.repo.ListFields(ctx, formID, true)
	answers, _ := s.repo.AnswersFor(ctx, []int64{submissionID})
	files, _ := s.repo.FilesFor(ctx, []int64{submissionID})

	detail := dynamicform_dto.SubmissionDetail{
		SubmissionID: sub.SubmissionID, FormID: formID, RespondentEmail: sub.RespondentEmail,
		RespondentName: strDeref(sub.RespondentName), IsValid: sub.IsValid, FormVersion: sub.FormVersion,
		SubmittedDate: sub.SubmittedDate.Format(timeLayout), Answers: map[string]string{},
	}
	for fieldID, v := range answers[submissionID] {
		detail.Answers["field_"+strconv.FormatInt(fieldID, 10)] = v
	}
	for _, f := range fields {
		detail.Fields = append(detail.Fields, toFieldResponse(f))
	}
	for _, fl := range files[submissionID] {
		detail.Files = append(detail.Files, dynamicform_dto.SubmissionFileRef{
			FieldID: fl.FieldID, FileURL: fl.FileURL, OriginalFileName: fl.OriginalFileName,
		})
	}
	return detail, nil
}

func (s *ServiceImpl) UpdateSubmission(ctx context.Context, formID, submissionID int64, actorID int64, perms []string, req dynamicform_dto.EditSubmissionRequest, files map[int64]*multipart.FileHeader) error {
	form, err := s.getOwnedForm(ctx, formID, actorID, perms)
	if err != nil {
		return err
	}
	sub, err := s.repo.GetSubmission(ctx, formID, submissionID)
	if err != nil {
		return apperror.NotFound("Tanggapan tidak ditemukan")
	}
	fields, _ := s.repo.ListFields(ctx, formID, true)

	// Decode req answers into the string-slice shape validateAnswers expects.
	values := map[int64][]string{}
	for k, raw := range req.Answers {
		if !strings.HasPrefix(k, "field_") {
			continue
		}
		id, convErr := strconv.ParseInt(strings.TrimPrefix(k, "field_"), 10, 64)
		if convErr != nil {
			continue
		}
		values[id] = decodeToStrings(raw)
	}
	// File fields are optional on edit — treat every one as satisfied.
	fileSatisfied := map[int64]bool{}
	for _, f := range fields {
		if f.FieldType == "file" {
			fileSatisfied[f.FieldID] = true
		}
	}
	if errs := validateAnswers(fields, values, fileSatisfied); len(errs) > 0 {
		return apperror.Validation("Data tidak valid", errs)
	}

	answers := map[int64]string{}
	for _, f := range fields {
		if isDisplayType(f.FieldType) || f.FieldType == "file" {
			continue
		}
		if _, present := values[f.FieldID]; !present {
			continue
		}
		if f.FieldType == "checkbox" {
			b, _ := json.Marshal(allVals(values, f.FieldID))
			answers[f.FieldID] = string(b)
			continue
		}
		answers[f.FieldID] = firstVal(values, f.FieldID)
	}

	existingFiles, _ := s.repo.FilesFor(ctx, []int64{submissionID})
	oldByField := map[int64]string{}
	for _, fl := range existingFiles[submissionID] {
		oldByField[fl.FieldID] = fl.FileURL
	}

	replaced := map[int64]dynamicform_model.File{}
	for fieldID, fh := range files {
		if fh == nil {
			continue
		}
		url, saveErr := s.saveFieldFile(fh)
		if saveErr != nil {
			return apperror.Unprocessable(saveErr.Error())
		}
		size := int(fh.Size / 1024)
		mt := mimeOf(fh.Filename)
		replaced[fieldID] = dynamicform_model.File{
			FileURL: url, OriginalFileName: fh.Filename, MimeType: &mt, FileSizeKB: &size,
		}
	}

	// Mirror respondent email/name if the corresponding field changed.
	email := sub.RespondentEmail
	if sysF, ok := systemEmailField(fields); ok {
		if v := firstVal(values, sysF.FieldID); v != "" {
			if _, e := mail.ParseAddress(v); e == nil {
				email = v
			}
		}
	}
	name := sub.RespondentName
	if guessed := guessRespondentName(fields, values); guessed != nil {
		name = guessed
	}

	if err := s.repo.EditSubmission(ctx, dynamicform_repository.EditData{
		FormID: formID, SubmissionID: submissionID, RespondentEmail: email, RespondentName: name,
		Answers: answers, ReplacedFiles: replaced,
	}); err != nil {
		return apperror.Internal("Gagal menyimpan perubahan")
	}
	for fieldID := range replaced {
		if old, ok := oldByField[fieldID]; ok && old != "" {
			_ = s.uploader.DeleteFile(old)
		}
	}
	s.logAudit(ctx, actorID, formID, "edit_response", nil, answers, map[string]int64{"submissionID": submissionID})

	if form.GsheetEnabled && form.GsheetSpreadsheetID != nil && s.gsheet.Enabled() {
		_, _ = s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
			Queue: jobqueue_model.QueueDefault, JobType: constants.JobDynamicFormGSheetUpdate,
			Payload: map[string]int64{"submissionID": submissionID},
		})
	}
	return nil
}

func (s *ServiceImpl) DeleteSubmission(ctx context.Context, formID, submissionID int64, actorID int64, perms []string) error {
	form, err := s.getOwnedForm(ctx, formID, actorID, perms)
	if err != nil {
		return err
	}
	sub, err := s.repo.GetSubmission(ctx, formID, submissionID)
	if err != nil {
		return apperror.NotFound("Tanggapan tidak ditemukan")
	}
	rowIndex := 0
	if sub.GsheetRowIndex != nil {
		rowIndex = *sub.GsheetRowIndex
	}
	urls, _, err := s.repo.DeleteSubmission(ctx, formID, submissionID)
	if err != nil {
		return apperror.Internal("")
	}
	for _, u := range urls {
		_ = s.uploader.DeleteFile(u)
	}
	s.logAudit(ctx, actorID, formID, "delete_response", nil, nil, map[string]int64{"submissionID": submissionID})

	if form.GsheetEnabled && form.GsheetSpreadsheetID != nil && s.gsheet.Enabled() {
		_, _ = s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
			Queue: jobqueue_model.QueueDefault, JobType: constants.JobDynamicFormGSheetDelete,
			Payload: map[string]int64{"formID": formID, "submissionID": submissionID, "gsheetRowIndex": int64(rowIndex)},
		})
	}
	return nil
}

func (s *ServiceImpl) DeleteResponses(ctx context.Context, formID int64, actorID int64, perms []string) error {
	form, err := s.getOwnedForm(ctx, formID, actorID, perms)
	if err != nil {
		return err
	}
	urls, err := s.repo.DeleteAllSubmissions(ctx, formID)
	if err != nil {
		return apperror.Internal("")
	}
	for _, u := range urls {
		_ = s.uploader.DeleteFile(u)
	}
	s.logAudit(ctx, actorID, formID, "delete_responses", nil, nil, nil)

	if form.GsheetEnabled && form.GsheetSpreadsheetID != nil && s.gsheet.Enabled() {
		_, _ = s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
			Queue: jobqueue_model.QueueDefault, JobType: constants.JobDynamicFormGSheetRebuild,
			Payload: map[string]int64{"formID": formID},
		})
	}
	return nil
}

func (s *ServiceImpl) ExportCSV(ctx context.Context, formID int64, actorID int64, perms []string, w io.Writer) (string, error) {
	form, err := s.getOwnedForm(ctx, formID, actorID, perms)
	if err != nil {
		return "", err
	}
	fields, _ := s.repo.ListFields(ctx, formID, true)
	subs, _, err := s.repo.ListSubmissions(ctx, formID, dynamicform_dto.SubmissionFilter{Limit: 100000, Offset: 0})
	if err != nil {
		return "", apperror.Internal("")
	}
	ids := make([]int64, len(subs))
	for i, sub := range subs {
		ids[i] = sub.SubmissionID
	}
	answers, _ := s.repo.AnswersFor(ctx, ids)

	cw := csv.NewWriter(w)
	if err := cw.Write(buildHeader(fields)); err != nil {
		return "", apperror.Internal("")
	}
	for _, sub := range subs {
		if err := cw.Write(buildRow(sub, answers[sub.SubmissionID], fields)); err != nil {
			return "", apperror.Internal("")
		}
	}
	cw.Flush()
	return form.Slug, cw.Error()
}

// ---------------------------------------------------------------------------
// analytics
// ---------------------------------------------------------------------------

var chartFieldTypes = map[string]bool{
	"radio": true, "dropdown": true, "checkbox": true, "linear_scale": true,
	"rating": true, "number": true, "date": true, "email": true, "short_text": true,
}

func (s *ServiceImpl) GetAnalytics(ctx context.Context, formID int64, actorID int64, perms []string) (dynamicform_dto.Analytics, error) {
	if _, err := s.getOwnedForm(ctx, formID, actorID, perms); err != nil {
		return dynamicform_dto.Analytics{}, err
	}
	out := dynamicform_dto.Analytics{}

	since := time.Now().AddDate(0, 0, -29)
	perDay, _ := s.repo.SubmissionsPerDay(ctx, formID, time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, time.Local))
	for i := 0; i < 30; i++ {
		day := since.AddDate(0, 0, i).Format("2006-01-02")
		out.SubmissionsPerDay = append(out.SubmissionsPerDay, dynamicform_dto.DayCount{Date: day, Count: perDay[day]})
	}

	out.ValidCount, out.InvalidCount, _ = s.repo.ValidCounts(ctx, formID)
	out.TotalFiles, _ = s.repo.TotalFiles(ctx, formID)

	recent, _ := s.repo.RecentSubmissions(ctx, formID, 10)
	for _, r := range recent {
		out.Recent = append(out.Recent, dynamicform_dto.RecentSubmission{
			SubmissionID: r.SubmissionID, RespondentEmail: r.RespondentEmail,
			RespondentName: strDeref(r.RespondentName), IsValid: r.IsValid,
			SubmittedDate: r.SubmittedDate.Format(timeLayout),
		})
	}

	fields, _ := s.repo.ListFields(ctx, formID, true)
	for _, f := range fields {
		if !chartFieldTypes[f.FieldType] {
			continue
		}
		rows, _ := s.repo.AnswerValueCounts(ctx, formID, f.FieldID)
		out.FieldCharts = append(out.FieldCharts, buildFieldChart(f, rows))
	}
	return out, nil
}

func buildFieldChart(f dynamicform_model.Field, rows []dynamicform_repository.ValueCount) dynamicform_dto.FieldChart {
	tally := map[string]int{}
	for _, r := range rows {
		switch f.FieldType {
		case "checkbox":
			var arr []string
			if json.Unmarshal([]byte(r.Value), &arr) == nil {
				for _, a := range arr {
					tally[a] += r.Count
				}
			} else {
				tally[r.Value] += r.Count
			}
		case "email":
			at := strings.LastIndex(r.Value, "@")
			if at >= 0 && at < len(r.Value)-1 {
				tally["@"+r.Value[at+1:]] += r.Count
			} else {
				tally[r.Value] += r.Count
			}
		default:
			tally[r.Value] += r.Count
		}
	}

	buckets := make([]dynamicform_dto.ChartBucket, 0, len(tally))
	for label, count := range tally {
		buckets = append(buckets, dynamicform_dto.ChartBucket{Label: label, Count: count})
	}

	numeric := f.FieldType == "number" || f.FieldType == "linear_scale" || f.FieldType == "rating"
	if numeric {
		sort.Slice(buckets, func(i, j int) bool {
			return parseNum(buckets[i].Label) < parseNum(buckets[j].Label)
		})
	} else {
		sort.Slice(buckets, func(i, j int) bool { return buckets[i].Count > buckets[j].Count })
	}

	chartType := "bar"
	switch f.FieldType {
	case "radio", "dropdown":
		chartType = "doughnut"
	case "email", "short_text", "date":
		chartType = "bar-horizontal"
		if len(buckets) > 10 {
			buckets = buckets[:10]
		}
	}
	return dynamicform_dto.FieldChart{
		FieldID: f.FieldID, Label: f.Label, FieldType: f.FieldType, ChartType: chartType, Buckets: buckets,
	}
}

func parseNum(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// decodeToStrings turns a raw JSON answer value into a []string (a JSON array
// stays a slice; a scalar becomes a one-element slice).
func decodeToStrings(raw json.RawMessage) []string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var arr []any
		if json.Unmarshal(raw, &arr) == nil {
			out := make([]string, 0, len(arr))
			for _, v := range arr {
				out = append(out, fmt.Sprint(v))
			}
			return out
		}
	}
	var scalar any
	if json.Unmarshal(raw, &scalar) == nil {
		return []string{fmt.Sprint(scalar)}
	}
	return []string{trimmed}
}
