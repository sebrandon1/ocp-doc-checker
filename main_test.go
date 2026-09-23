package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sebrandon1/ocp-doc-checker/pkg/checker"
)

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

	encoded, err := json.Marshal(makeSingleJSONResult(result))
	if err != nil {
		t.Fatalf("marshal single result: %v", err)
	}
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

	encoded, err := json.Marshal(makeBatchJSONResult(results))
	if err != nil {
		t.Fatalf("marshal batch result: %v", err)
	}
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
	encoded, err := json.Marshal(makeBatchJSONResult(nil))
	if err != nil {
		t.Fatalf("marshal empty batch result: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal empty batch result: %v", err)
	}
	if got := string(decoded["results"]); got != "[]" {
		t.Errorf("results = %s, want []", got)
	}
}
