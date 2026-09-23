package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
		t.Errorf("error count = %d with %d errors, want 2", batch.ErrorCount, len(batch.Errors))
	}
	if got := string(batch.Errors[1].URL); strings.Contains(got, "secret") || strings.Contains(got, "private") {
		t.Errorf("error URL leaked credentials or query data: %q", got)
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

	results, checkErrors := checkURLLocations(c, locations, nil)
	encoded := encodeBatchForTest(t, results, checkErrors)
	var batch batchCheckJSONOutput
	if err := json.Unmarshal(encoded, &batch); err != nil {
		t.Fatalf("unmarshal batch output: %v", err)
	}

	if batch.TotalCount != 1 || batch.UptodateCount != 1 || batch.OutdatedCount != 0 {
		t.Errorf("successful counts = total %d, up-to-date %d, outdated %d; want 1, 1, 0", batch.TotalCount, batch.UptodateCount, batch.OutdatedCount)
	}
	if batch.ErrorCount != 1 || len(batch.Errors) != 1 {
		t.Errorf("error count = %d with %d errors, want 1", batch.ErrorCount, len(batch.Errors))
	}
	if len(batch.Results) != 1 || batch.Results[0].IsOutdated {
		t.Errorf("successful results = %#v, want one up-to-date result", batch.Results)
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
	rawURL := "https://user:secret@example.invalid/path?token=private"
	checkError := sanitizeBatchCheckError(rawURL, errors.New("request failed for "+rawURL+" and https://api.example.invalid/?key=hidden"))

	for _, value := range []string{"secret", "private", "hidden"} {
		if strings.Contains(checkError.URL, value) || strings.Contains(checkError.Message, value) {
			t.Errorf("sanitized error leaked %q: %+v", value, checkError)
		}
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
