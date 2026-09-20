package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestClientSignsRequests(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/prefix/admin/user" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.URL.Query().Get("format") != "json" || request.URL.Query().Get("uid") != "alice" {
			t.Errorf("query = %v", request.URL.Query())
		}
		if got := request.Header.Get("Authorization"); !strings.HasPrefix(got, "AWS4-HMAC-SHA256 Credential=access-key/") ||
			!strings.Contains(got, "/default/s3/aws4_request") {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("X-Amz-Date"); got != "20260920T120000Z" {
			t.Errorf("X-Amz-Date = %q", got)
		}
		if got := request.Header.Get("User-Agent"); got != "admin-test" {
			t.Errorf("User-Agent = %q", got)
		}
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := NewClient(
		server.URL+"/prefix",
		"access-key",
		"secret-key",
		WithUserAgent("admin-test"),
	)
	if err != nil {
		t.Fatal(err)
	}
	client.now = func() time.Time {
		return time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)
	}
	request, err := client.newRequest(context.Background(), http.MethodGet, "user", url.Values{"uid": {"alice"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.do(request); err != nil {
		t.Fatal(err)
	}
}

func TestClientReturnsStructuredAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte(`{"Code":"NoSuchUser","RequestId":"request-id","HostId":"host-id"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "access-key", "secret-key")
	if err != nil {
		t.Fatal(err)
	}
	request, err := client.newRequest(context.Background(), http.MethodGet, "user", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.do(request)
	var apiError *APIError
	if !errors.As(err, &apiError) || !errors.Is(err, ErrNoSuchUser) {
		t.Fatalf("error = %v, want NoSuchUser APIError", err)
	}
	if apiError.StatusCode != http.StatusNotFound || apiError.RequestID != "request-id" || apiError.HostID != "host-id" {
		t.Fatalf("API error = %#v", apiError)
	}
}

func TestClientRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", maxResponseBody+1)))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "access-key", "secret-key")
	if err != nil {
		t.Fatal(err)
	}
	request, err := client.newRequest(context.Background(), http.MethodGet, "user", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.do(request); err == nil || !strings.Contains(err.Error(), "response exceeds") {
		t.Fatalf("error = %v, want response size error", err)
	}
}

func TestNewClientValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		endpoint  string
		accessKey string
		secretKey string
		options   []Option
	}{
		{name: "missing scheme", endpoint: "rgw.example", accessKey: "access", secretKey: "secret"},
		{name: "unsupported scheme", endpoint: "ftp://rgw.example", accessKey: "access", secretKey: "secret"},
		{name: "query", endpoint: "https://rgw.example?x=1", accessKey: "access", secretKey: "secret"},
		{name: "missing access key", endpoint: "https://rgw.example", secretKey: "secret"},
		{name: "missing secret key", endpoint: "https://rgw.example", accessKey: "access"},
		{name: "nil HTTP client", endpoint: "https://rgw.example", accessKey: "access", secretKey: "secret", options: []Option{WithHTTPClient(nil)}},
		{name: "empty admin path", endpoint: "https://rgw.example", accessKey: "access", secretKey: "secret", options: []Option{WithAdminPath("/")}},
		{name: "empty region", endpoint: "https://rgw.example", accessKey: "access", secretKey: "secret", options: []Option{WithRegion(" ")}},
		{name: "nil option", endpoint: "https://rgw.example", accessKey: "access", secretKey: "secret", options: []Option{nil}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewClient(test.endpoint, test.accessKey, test.secretKey, test.options...); err == nil {
				t.Fatal("error = nil, want validation error")
			}
		})
	}
}
