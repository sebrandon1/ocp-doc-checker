package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/sebrandon1/ocp-doc-checker/pkg/checker"
)

func TestBatchJSONReportsAllFailedChecks(t *testing.T) {
	c := checker.NewChecker()
	c.SetVersions([]string{})
	locations := []URLLocation{
		{URL: "not a URL"},
		{URL: "https://user:secret@example.invalid/path?token=private"},
	}

	results, checkErrors := checkURLLocations(c, locations, nil)
	encoded := encodeBatchForTest(t, results, checkErrors)
	var batch batchCheckJSONOutput
	if err := json.Unmarshal(encoded, &batch); err != nil {
		t.Fatalf("unmarshal batch output: %v", err)
	}

	if len(results) != 0 || batch.TotalCount != 0 || batch.UptodateCount != 0 || batch.OutdatedCount != 0 {
		t.Errorf("failed checks affected successful result counts: results=%d batch=%+v", len(results), batch)
	}
	if batch.ErrorCount != 2 || len(batch.Errors) != 2 {
		t.Fatalf("error count = %d with %d errors, want 2", batch.ErrorCount, len(batch.Errors))
	}
	if len(batch.Errors) > 1 {
		if got := string(batch.Errors[1].URL); strings.Contains(got, "secret") || strings.Contains(got, "private") {
			t.Errorf("error URL leaked credentials or query data: %q", got)
		}
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "private") {
		t.Errorf("batch JSON leaked credentials or query data: %s", encoded)
	}
}

func TestBatchJSONSeparatesMixedSuccessAndFailure(t *testing.T) {
	c := checker.NewChecker()
	c.SetVersions([]string{})
	locations := []URLLocation{
		{URL: "https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html-single/test/index"},
		{URL: "https://user:secret@example.invalid/path?token=private"},
	}

	var checkedURLs []string
	var checkedIndexes []int
	results, checkErrors := checkURLLocations(c, locations, func(index int, location URLLocation) {
		checkedIndexes = append(checkedIndexes, index)
		checkedURLs = append(checkedURLs, location.URL)
	})
	encoded := encodeBatchForTest(t, results, checkErrors)
	var batch batchCheckJSONOutput
	if err := json.Unmarshal(encoded, &batch); err != nil {
		t.Fatalf("unmarshal batch output: %v", err)
	}

	if batch.TotalCount != 1 || batch.UptodateCount != 1 || batch.OutdatedCount != 0 {
		t.Errorf("successful counts = total %d, up-to-date %d, outdated %d; want 1, 1, 0", batch.TotalCount, batch.UptodateCount, batch.OutdatedCount)
	}
	if batch.ErrorCount != 1 || len(batch.Errors) != 1 || len(checkErrors) != 1 {
		t.Fatalf("error count = %d with %d output errors and %d collected errors, want 1", batch.ErrorCount, len(batch.Errors), len(checkErrors))
	}
	if len(batch.Results) != 1 || batch.Results[0].IsOutdated {
		t.Errorf("successful results = %#v, want one up-to-date result", batch.Results)
	}
	if !reflect.DeepEqual(checkedURLs, []string{locations[0].URL, locations[1].URL}) || !reflect.DeepEqual(checkedIndexes, []int{0, 1}) {
		t.Errorf("before-check callback received indexes %v and URLs %v, want both URLs in order", checkedIndexes, checkedURLs)
	}

	var textOutput bytes.Buffer
	if err := writeBatchTextResults(&textOutput, results, checkErrors, false); err != nil {
		t.Fatalf("write batch text output: %v", err)
	}
	for _, want := range []string{
		"Summary: successful=1, up-to-date=1, outdated=0, errors=1",
		"❌ Failed URL Checks:",
		checkErrors[0].URL + ": " + checkErrors[0].Message,
	} {
		if !strings.Contains(textOutput.String(), want) {
			t.Errorf("text output is missing %q:\n%s", want, textOutput.String())
		}
	}
}

func TestBatchJSONKeepsOutdatedResultsSeparateFromErrors(t *testing.T) {
	results := []*checker.CheckResult{{
		OriginalURL:     "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/test/index",
		OriginalVersion: "4.17",
		LatestVersion:   "4.18",
		IsOutdated:      true,
		NewerVersions: []checker.VersionCheckResult{{
			Version: "4.18",
			URL:     "https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html-single/test/index",
		}},
	}}
	checkErrors := []batchCheckError{{URL: "[redacted URL]", Message: "network request failed"}}

	encoded := encodeBatchForTest(t, results, checkErrors)
	var batch batchCheckJSONOutput
	if err := json.Unmarshal(encoded, &batch); err != nil {
		t.Fatalf("unmarshal batch output: %v", err)
	}
	if batch.TotalCount != 1 || batch.UptodateCount != 0 || batch.OutdatedCount != 1 || batch.ErrorCount != 1 {
		t.Errorf("counts = total %d, up-to-date %d, outdated %d, errors %d; want 1, 0, 1, 1", batch.TotalCount, batch.UptodateCount, batch.OutdatedCount, batch.ErrorCount)
	}
	if len(batch.Results) != 1 || len(batch.Results[0].NewerVersions) != 1 || batch.Results[0].NewerVersions[0].Version != "4.18" {
		t.Errorf("newer versions were not preserved in batch output: %#v", batch.Results)
	}
}

func TestBatchTextSummarizesOutdatedResultsWithoutErrors(t *testing.T) {
	result := &checker.CheckResult{
		OriginalURL:     "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/test/index",
		OriginalVersion: "4.17",
		LatestVersion:   "4.19",
		IsOutdated:      true,
		NewerVersions: []checker.VersionCheckResult{
			{Version: "4.18", URL: "https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html-single/test/index"},
			{Version: "4.19", URL: "https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/test/index"},
		},
	}
	var output bytes.Buffer
	if err := writeBatchTextResults(&output, []*checker.CheckResult{result}, nil, false); err != nil {
		t.Fatalf("write batch text output: %v", err)
	}

	for _, want := range []string{
		"Summary: successful=1, up-to-date=0, outdated=1, errors=0",
		"2 newer versions available",
		"🔧 Recommended Updates:",
		"Old: " + result.OriginalURL,
		"New: " + result.NewerVersions[1].URL,
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("text output is missing %q:\n%s", want, output.String())
		}
	}
}

func TestBatchTextListsAllNewerVersionsWhenRequested(t *testing.T) {
	previous := *allAvailableFlag
	*allAvailableFlag = true
	t.Cleanup(func() { *allAvailableFlag = previous })
	result := &checker.CheckResult{
		OriginalURL:     "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/test/index",
		OriginalVersion: "4.17",
		LatestVersion:   "4.19",
		IsOutdated:      true,
		NewerVersions: []checker.VersionCheckResult{
			{Version: "4.18", URL: "https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html-single/test/index"},
			{Version: "4.19", URL: "https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/test/index"},
		},
	}
	var output bytes.Buffer
	if err := writeBatchTextResults(&output, []*checker.CheckResult{result}, nil, false); err != nil {
		t.Fatalf("write batch text output: %v", err)
	}
	for _, want := range []string{"Available newer versions:", "Version 4.18", "Version 4.19"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("text output is missing %q:\n%s", want, output.String())
		}
	}
}

func TestBatchTextReturnsWriterErrors(t *testing.T) {
	wantErr := errors.New("write failed")
	writeErr := writeBatchTextResults(batchWriterFunc(func([]byte) (int, error) {
		return 0, wantErr
	}), nil, nil, false)
	if !errors.Is(writeErr, wantErr) {
		t.Errorf("write error = %v, want %v", writeErr, wantErr)
	}
}

func TestBatchJSONReportsNoErrorsAsEmptyArray(t *testing.T) {
	c := checker.NewChecker()
	c.SetVersions([]string{})
	locations := []URLLocation{{URL: "https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html-single/test/index"}}

	results, checkErrors := checkURLLocations(c, locations, nil)
	encoded := encodeBatchForTest(t, results, checkErrors)
	var batch batchCheckJSONOutput
	if err := json.Unmarshal(encoded, &batch); err != nil {
		t.Fatalf("unmarshal batch output: %v", err)
	}
	if batch.ErrorCount != 0 || batch.Errors == nil || len(batch.Errors) != 0 {
		t.Errorf("errors = %#v with count %d, want an empty array and zero count", batch.Errors, batch.ErrorCount)
	}
}

func TestSanitizeBatchCheckErrorRedactsURLsInMessage(t *testing.T) {
	rawURL := "https://user:secret@example.invalid/path?token=private#access_token=fragment-secret"
	checkError := sanitizeBatchCheckError(rawURL, errors.New("request failed for "+rawURL+" and https://api.example.invalid/?key=hidden"))

	for _, value := range []string{"secret", "private", "hidden", "fragment-secret"} {
		if strings.Contains(checkError.URL, value) || strings.Contains(checkError.Message, value) {
			t.Errorf("sanitized error leaked %q: %+v", value, checkError)
		}
	}

	malformedURL := "https://example.invalid/%zz"
	malformedError := sanitizeBatchCheckError(malformedURL, errors.New("parse failed for "+malformedURL))
	if malformedError.URL != "[redacted URL]" {
		t.Errorf("malformed URL was not fully redacted: %+v", malformedError)
	}
}

func encodeBatchForTest(t *testing.T, results []*checker.CheckResult, checkErrors []batchCheckError) []byte {
	t.Helper()
	var output bytes.Buffer
	if err := encodeBatchCheckResults(&output, results, checkErrors); err != nil {
		t.Fatalf("encode batch JSON: %v", err)
	}
	if !json.Valid(output.Bytes()) {
		t.Fatalf("batch output is invalid JSON: %s", output.Bytes())
	}
	return output.Bytes()
}

type batchWriterFunc func([]byte) (int, error)

func (write batchWriterFunc) Write(data []byte) (int, error) {
	return write(data)
}

func TestSingleJSONResultEscapesValuesAndPreservesFields(t *testing.T) {
	result := &checker.CheckResult{
		OriginalURL:     "https://example.com/\"quoted\"\\path\nnext",
		OriginalVersion: "4.17\tβ",
		LatestVersion:   "4.19",
		IsOutdated:      true,
		NewerVersions: []checker.VersionCheckResult{{
			Version: "4.19",
			URL:     "https://example.com/new?query=\"value\"&path=\\",
		}},
	}

	var output bytes.Buffer
	if err := encodeSingleJSONResult(&output, result); err != nil {
		t.Fatalf("encode single result: %v", err)
	}
	encoded := output.Bytes()
	if !json.Valid(encoded) {
		t.Fatalf("single result is invalid JSON: %s", encoded)
	}

	var decoded singleJSONOutput
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal single result: %v", err)
	}
	if want := makeSingleJSONResult(result); !reflect.DeepEqual(decoded, want) {
		t.Errorf("decoded result = %#v, want %#v", decoded, want)
	}
}

func TestBatchJSONResultPreservesShapeAndCounts(t *testing.T) {
	results := []*checker.CheckResult{
		{OriginalURL: "https://example.com/\"one\"", OriginalVersion: "4.17", LatestVersion: "4.17"},
		{OriginalURL: "https://example.com/two\\path", OriginalVersion: "4.17", LatestVersion: "4.19", IsOutdated: true},
	}

	var output bytes.Buffer
	if err := encodeBatchJSONResult(&output, results); err != nil {
		t.Fatalf("encode batch result: %v", err)
	}
	encoded := output.Bytes()
	if !json.Valid(encoded) {
		t.Fatalf("batch result is invalid JSON: %s", encoded)
	}

	var decoded batchJSONResult
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal batch result: %v", err)
	}
	if decoded.TotalCount != 2 || decoded.UptodateCount != 1 || decoded.OutdatedCount != 1 {
		t.Errorf("counts = (%d, %d, %d), want (2, 1, 1)", decoded.TotalCount, decoded.UptodateCount, decoded.OutdatedCount)
	}
	if want := makeBatchJSONResult(results); !reflect.DeepEqual(decoded, want) {
		t.Errorf("decoded result = %#v, want %#v", decoded, want)
	}
}

func TestBatchJSONResultUsesEmptyArrayForNoResults(t *testing.T) {
	var output bytes.Buffer
	if err := encodeBatchJSONResult(&output, nil); err != nil {
		t.Fatalf("encode empty batch result: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal empty batch result: %v", err)
	}
	if got := string(decoded["results"]); got != "[]" {
		t.Errorf("results = %s, want []", got)
	}
}
