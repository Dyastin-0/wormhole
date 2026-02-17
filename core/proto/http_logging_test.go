package proto

import (
	"net/http"
	"testing"
)

func TestHTTPLogRoundTrip(t *testing.T) {
	original := &HTTPLog{
		Timestamp: 1700000000,
		Duration:  42_000,
		Status:    200,
		Method:    "POST",
		Path:      "/api/v1/resource",
		ReqBody:   []byte(`{"hello":"world"}`),
		RespBody:  []byte(`{"ok":true}`),
		ReqHeaders: http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer token123"},
		},
		RespHeaders: http.Header{
			"Content-Type": {"application/json"},
			"X-Request-Id": {"abc-123"},
		},
	}

	data, err := SerializeHTTPLog(original)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	got, err := DeserializeHTTPLog(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if got.Timestamp != original.Timestamp {
		t.Errorf("Timestamp: got %d, want %d", got.Timestamp, original.Timestamp)
	}
	if got.Duration != original.Duration {
		t.Errorf("Duration: got %d, want %d", got.Duration, original.Duration)
	}
	if got.Status != original.Status {
		t.Errorf("Status: got %d, want %d", got.Status, original.Status)
	}
	if got.Method != original.Method {
		t.Errorf("Method: got %q, want %q", got.Method, original.Method)
	}
	if got.Path != original.Path {
		t.Errorf("Path: got %q, want %q", got.Path, original.Path)
	}
	if string(got.ReqBody) != string(original.ReqBody) {
		t.Errorf("ReqBody: got %q, want %q", got.ReqBody, original.ReqBody)
	}
	if string(got.RespBody) != string(original.RespBody) {
		t.Errorf("RespBody: got %q, want %q", got.RespBody, original.RespBody)
	}
	for key, wantVals := range original.ReqHeaders {
		gotVals, ok := got.ReqHeaders[key]
		if !ok {
			t.Errorf("ReqHeaders missing key %q", key)
			continue
		}
		for i, v := range wantVals {
			if gotVals[i] != v {
				t.Errorf("ReqHeaders[%q][%d]: got %q, want %q", key, i, gotVals[i], v)
			}
		}
	}
}

func TestHTTPLogEmpty(t *testing.T) {
	original := NewHTTPLog(1700000000, 1234)
	original.Status = 204
	original.Method = "DELETE"
	original.Path = "/item/99"

	data, err := SerializeHTTPLog(original)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	got, err := DeserializeHTTPLog(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if got.Method != original.Method {
		t.Errorf("Method: got %q, want %q", got.Method, original.Method)
	}
	if len(got.ReqHeaders) != 0 {
		t.Errorf("expected nil ReqHeaders, got %v", got.ReqHeaders)
	}
}
