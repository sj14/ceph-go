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

func TestGetBucket(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", request.Method)
		}
		if got := request.URL.EscapedPath(); got != "/dashboard/api/rgw/bucket/tenant%2Fphotos%202026" {
			t.Errorf("escaped path = %q, want encoded bucket path segment", got)
		}
		if got := request.URL.Query(); !reflect.DeepEqual(got, url.Values{"daemon_name": {"rgw.one"}}) {
			t.Errorf("query = %#v, want daemon_name=rgw.one", got)
		}
		if got := request.Header.Get("Accept"); got != defaultMediaType {
			t.Errorf("Accept = %q, want %q", got, defaultMediaType)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", got)
		}

		writer.Header().Set("Content-Type", defaultMediaType)
		_, _ = writer.Write([]byte(`{
			"bucket":"photos 2026",
			"tenant":"tenant",
			"bid":"tenant/photos 2026",
			"zonegroup":"default",
			"placement_rule":"default-placement",
			"explicit_placement":{"data_pool":"data","data_extra_pool":"extra","index_pool":"index"},
			"id":"bucket-id",
			"marker":"marker",
			"index_type":"Normal",
			"index_generation":2,
			"num_shards":11,
			"reshard_status":"not-resharding",
			"judge_reshard_lock_time":"",
			"object_lock_enabled":true,
			"mfa_enabled":false,
			"owner":"alice",
			"ver":"1#2",
			"master_ver":"1#2",
			"mtime":"2026-09-19T12:00:00Z",
			"creation_time":"2026-09-18T12:00:00Z",
			"max_marker":"",
			"usage":{"rgw.main":{"size":12,"size_actual":4096,"size_utilized":12,"size_kb":1,"size_kb_actual":4,"size_kb_utilized":1,"num_objects":3}},
			"bucket_quota":{"enabled":true,"check_on_raw":false,"max_size":1099511627776,"max_size_kb":1073741824,"max_objects":5000000000},
			"read_tracker":7,
			"encryption":"Enabled",
			"versioning":"Enabled",
			"mfa_delete":"Disabled",
			"bucket_policy":{"Version":"2012-10-17"},
			"acl":"<AccessControlPolicy/>",
			"replication":{"sync_policy_active":true,"replication_rules_configured":true,"policy":{"Rule":[{"ID":"copy"}]}},
			"lifecycle":null,
			"lifecycle_progress":[{"bucket":"photos 2026","status":"COMPLETE"}],
			"lock_enabled":true,
			"lock_mode":"COMPLIANCE",
			"lock_retention_period_days":365,
			"lock_retention_period_years":null
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/dashboard", WithBearerToken("secret-token"))
	if err != nil {
		t.Fatal(err)
	}
	bucket, err := client.GetBucket(context.Background(), GetBucketRequest{
		Name:       "tenant/photos 2026",
		DaemonName: "rgw.one",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != "photos 2026" || bucket.Tenant != "tenant" || bucket.BID != "tenant/photos 2026" {
		t.Errorf("bucket identity = %#v", bucket)
	}
	if bucket.Quota.MaxSize != 1099511627776 || bucket.Quota.MaxObjects != 5000000000 {
		t.Errorf("quota = %#v", bucket.Quota)
	}
	if bucket.Usage["rgw.main"].NumObjects != 3 {
		t.Errorf("usage = %#v", bucket.Usage)
	}
	if bucket.LockRetentionDays == nil || *bucket.LockRetentionDays != 365 {
		t.Errorf("lock retention days = %v, want 365", bucket.LockRetentionDays)
	}
	if bucket.LockRetentionYears != nil {
		t.Errorf("lock retention years = %v, want nil", bucket.LockRetentionYears)
	}
	if string(bucket.BucketPolicy) != `{"Version":"2012-10-17"}` {
		t.Errorf("bucket policy = %s", bucket.BucketPolicy)
	}
	if string(bucket.Replication.Policy) != `{"Rule":[{"ID":"copy"}]}` {
		t.Errorf("replication policy = %s", bucket.Replication.Policy)
	}
}

func TestGetBucketValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://ceph.example")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetBucket(context.Background(), GetBucketRequest{}); err == nil {
		t.Fatal("GetBucket() error = nil, want empty name error")
	}
	if _, err := client.GetBucket(nil, GetBucketRequest{Name: "photos"}); err == nil {
		t.Fatal("GetBucket() error = nil, want nil context error")
	}
}

func TestGetBucketReturnsDecodeError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("not JSON"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	bucket, err := client.GetBucket(context.Background(), GetBucketRequest{Name: "photos"})
	if err == nil || !reflect.DeepEqual(bucket, Bucket{}) {
		t.Fatalf("GetBucket() = %#v, %v; want decode error", bucket, err)
	}
}

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
	err = client.CreateBucket(context.Background(), CreateBucketRequest{
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
	if err := client.CreateBucket(context.Background(), CreateBucketRequest{Name: "photos", UID: "bob"}); err != nil {
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
	err = client.CreateBucket(context.Background(), CreateBucketRequest{Name: "photos", UID: "bob"})
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
			if err := client.CreateBucket(context.Background(), test.input); err == nil {
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
