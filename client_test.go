package rgw

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestCreateBucket(t *testing.T) {
	t.Parallel()

	days := int64(5)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", request.Method)
		}
		if request.URL.Path != "/dashboard/api/rgw/bucket" {
			t.Errorf("path = %q, want /dashboard/api/rgw/bucket", request.URL.Path)
		}
		if got := request.Header.Get("Accept"); got != defaultMediaType {
			t.Errorf("Accept = %q, want %q", got, defaultMediaType)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", got)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) != 0 {
			t.Errorf("body = %q, want empty", body)
		}

		wantQuery := url.Values{
			"bucket":                     {"backups"},
			"uid":                        {"alice"},
			"zonegroup":                  {"default"},
			"placement_target":           {"default-placement"},
			"lock_enabled":               {"true"},
			"lock_mode":                  {"COMPLIANCE"},
			"lock_retention_period_days": {"5"},
			"encryption_state":           {"true"},
			"encryption_type":            {"aws:kms"},
			"key_id":                     {"key-1"},
			"tags":                       {"<Tagging/>"},
			"bucket_policy":              {`{"Version":"2012-10-17"}`},
			"canned_acl":                 {"private"},
			"replication":                {"true"},
			"daemon_name":                {"rgw.one"},
		}
		if !reflect.DeepEqual(request.URL.Query(), wantQuery) {
			t.Errorf("query = %#v, want %#v", request.URL.Query(), wantQuery)
		}

		writer.Header().Set("Content-Type", defaultMediaType)
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte("null"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/dashboard/", WithBearerToken("secret-token"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.CreateBucket(context.Background(), CreateBucketRequest{
		Name:               "backups",
		UID:                "alice",
		Zonegroup:          "default",
		PlacementTarget:    "default-placement",
		LockEnabled:        true,
		LockMode:           LockModeCompliance,
		LockRetentionDays:  &days,
		EncryptionEnabled:  true,
		EncryptionType:     "aws:kms",
		KeyID:              "key-1",
		Tags:               "<Tagging/>",
		BucketPolicy:       `{"Version":"2012-10-17"}`,
		CannedACL:          "private",
		ReplicationEnabled: true,
		DaemonName:         "rgw.one",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", response.StatusCode)
	}
	if string(response.Body) != "null" {
		t.Errorf("body = %q, want null", response.Body)
	}
}

func TestCreateBucketMinimalRequestIncludesCephDefaults(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		want := url.Values{
			"bucket":           {"photos"},
			"uid":              {"bob"},
			"lock_enabled":     {"false"},
			"encryption_state": {"false"},
			"replication":      {"false"},
		}
		if !reflect.DeepEqual(request.URL.Query(), want) {
			t.Errorf("query = %#v, want %#v", request.URL.Query(), want)
		}
		writer.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateBucket(context.Background(), CreateBucketRequest{Name: "photos", UID: "bob"}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateBucketReturnsAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusConflict)
		_, _ = writer.Write([]byte(`{"detail":"bucket exists"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.CreateBucket(context.Background(), CreateBucketRequest{Name: "photos", UID: "bob"})
	if response == nil || response.StatusCode != http.StatusConflict {
		t.Fatalf("response = %#v, want status 409", response)
	}
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiError.Path != "/api/rgw/bucket" {
		t.Errorf("error path = %q, want /api/rgw/bucket", apiError.Path)
	}
}

func TestCreateBucketValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://ceph.example")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		input CreateBucketRequest
	}{
		{name: "missing bucket", input: CreateBucketRequest{UID: "alice"}},
		{name: "missing uid", input: CreateBucketRequest{Name: "bucket"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := client.CreateBucket(context.Background(), test.input); err == nil {
				t.Fatal("CreateBucket() error = nil, want validation error")
			}
		})
	}
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	t.Parallel()

	for _, baseURL := range []string{"ceph.example", "ftp://ceph.example", "https://ceph.example?x=1"} {
		if _, err := NewClient(baseURL); err == nil {
			t.Errorf("NewClient(%q) error = nil, want error", baseURL)
		}
	}
}
