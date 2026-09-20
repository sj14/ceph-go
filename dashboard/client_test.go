package dashboard

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientAddsRequestMetadata(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Accept"); got != mediaTypeV1_0 {
			t.Errorf("Accept = %q, want %q", got, mediaTypeV1_0)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", got)
		}
		if got := request.Header.Get("User-Agent"); got != "rgw-go-test" {
			t.Errorf("User-Agent = %q, want rgw-go-test", got)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithBearerToken("secret-token"), WithUserAgent("rgw-go-test"))
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, client.endpoint("api/test").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.do(request); err != nil {
		t.Fatal(err)
	}
}

func TestClientPreservesRequestMediaType(t *testing.T) {
	t.Parallel()

	const mediaType = "application/vnd.ceph.api.v1.1+json"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Accept"); got != mediaType {
			t.Errorf("Accept = %q, want %q", got, mediaType)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, client.endpoint("api/test").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Accept", mediaType)
	if _, err := client.do(request); err != nil {
		t.Fatal(err)
	}
}

func TestClientReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusConflict)
		_, _ = writer.Write([]byte(`{"detail":"already exists"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, client.endpoint("api/test").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.do(request)
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusConflict || apiError.Method != http.MethodPost || apiError.Path != "/dashboard/api/test" {
		t.Errorf("API error metadata = %#v", apiError)
	}
	if apiError.Body != `{"detail":"already exists"}` {
		t.Errorf("API error body = %q", apiError.Body)
	}
}

func TestClientRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", maxResponseBody+1)))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, client.endpoint("api/test").String(), nil)
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
		name    string
		baseURL string
		options []Option
	}{
		{name: "missing scheme", baseURL: "ceph.example"},
		{name: "unsupported scheme", baseURL: "ftp://ceph.example"},
		{name: "query", baseURL: "https://ceph.example?x=1"},
		{name: "fragment", baseURL: "https://ceph.example#x"},
		{name: "empty bearer token", baseURL: "https://ceph.example", options: []Option{WithBearerToken(" ")}},
		{name: "nil HTTP client", baseURL: "https://ceph.example", options: []Option{WithHTTPClient(nil)}},
		{name: "nil option", baseURL: "https://ceph.example", options: []Option{nil}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewClient(test.baseURL, test.options...); err == nil {
				t.Fatal("error = nil, want validation error")
			}
		})
	}
}
