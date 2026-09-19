package rgw

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestListUsers(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", request.Method)
		}
		if request.URL.Path != "/dashboard/api/rgw/user" {
			t.Errorf("path = %q, want /dashboard/api/rgw/user", request.URL.Path)
		}
		wantQuery := url.Values{"daemon_name": {"rgw.one"}, "detailed": {"true"}}
		if got := request.URL.Query(); !reflect.DeepEqual(got, wantQuery) {
			t.Errorf("query = %#v, want %#v", got, wantQuery)
		}
		if got := request.Header.Get("Accept"); got != defaultMediaType {
			t.Errorf("Accept = %q, want %q", got, defaultMediaType)
		}

		writer.Header().Set("Content-Type", defaultMediaType)
		_, _ = writer.Write([]byte(`[
			{"uid":"alice","full_user_id":"tenant$alice","tenant":"tenant","user_id":"alice","display_name":"Alice","suspended":0,"max_buckets":1000},
			{"uid":"bob","full_user_id":"bob","user_id":"bob","display_name":"Bob","suspended":1,"max_buckets":-1}
		]`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	users, err := client.ListUsers(context.Background(), ListUsersRequest{DaemonName: "rgw.one"})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].UID != "alice" || users[0].FullUserID != "tenant$alice" || users[1].Suspended != 1 {
		t.Errorf("users = %#v", users)
	}
}

func TestGetUser(t *testing.T) {
	t.Parallel()

	stats := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", request.Method)
		}
		if got := request.URL.EscapedPath(); got != "/dashboard/api/rgw/user/tenant$alice%2Fadmin" {
			t.Errorf("escaped path = %q, want encoded user path segment", got)
		}
		wantQuery := url.Values{"daemon_name": {"rgw.one"}, "stats": {"false"}}
		if got := request.URL.Query(); !reflect.DeepEqual(got, wantQuery) {
			t.Errorf("query = %#v, want %#v", got, wantQuery)
		}

		writer.Header().Set("Content-Type", defaultMediaType)
		_, _ = writer.Write([]byte(`{
			"uid":"tenant$alice/admin",
			"full_user_id":"tenant$alice/admin",
			"tenant":"tenant",
			"user_id":"alice/admin",
			"display_name":"Alice",
			"email":"alice@example.com",
			"suspended":0,
			"max_buckets":5000000000,
			"subusers":[{"id":"tenant$alice/admin:swift","permissions":"read-write"}],
			"keys":[{"user":"tenant$alice/admin","access_key":"access","secret_key":"secret","active":true,"create_date":"2026-09-19T12:00:00Z"}],
			"swift_keys":[{"user":"tenant$alice/admin:swift","secret_key":"swift-secret","active":true}],
			"caps":[{"type":"users","perm":"read"}],
			"op_mask":"read, write, delete",
			"system":false,
			"admin":false,
			"default_placement":"default-placement",
			"default_storage_class":"STANDARD",
			"placement_tags":["ssd"],
			"bucket_quota":{"enabled":true,"check_on_raw":false,"max_size":1099511627776,"max_size_kb":1073741824,"max_objects":5000000000},
			"user_quota":{"enabled":false,"check_on_raw":false,"max_size":-1,"max_size_kb":0,"max_objects":-1},
			"temp_url_keys":[{"key":0,"val":"temporary"}],
			"type":"rgw",
			"mfa_ids":["mfa-1"],
			"account_id":"RGW123",
			"path":"/engineering/",
			"create_date":"2026-09-19T12:00:00Z",
			"tags":[{"key":"team","val":"storage"}],
			"group_ids":["group-1"],
			"managed_user_policies":["arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	user, err := client.GetUser(context.Background(), GetUserRequest{
		UID:        "tenant$alice/admin",
		DaemonName: "rgw.one",
		Stats:      &stats,
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.UID != "tenant$alice/admin" || user.MaxBuckets != 5000000000 {
		t.Errorf("user identity = %#v", user)
	}
	if len(user.Keys) != 1 || user.Keys[0].AccessKey != "access" {
		t.Errorf("keys = %#v", user.Keys)
	}
	if user.BucketQuota.MaxSize != 1099511627776 || user.BucketQuota.MaxObjects != 5000000000 {
		t.Errorf("bucket quota = %#v", user.BucketQuota)
	}
	if len(user.Tags) != 1 || user.Tags[0].Value != "storage" {
		t.Errorf("tags = %#v", user.Tags)
	}
	if user.Stats != nil {
		t.Errorf("stats = %#v, want nil", user.Stats)
	}
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	email := ""
	maxBuckets := int64(-1)
	system := false
	suspended := true
	generateKey := false
	accessKey := "access"
	secretKey := "secret"
	accountID := "RGW123"
	accountRoot := false
	policies := UserAccountPolicyChanges{
		Attach: []string{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
		Detach: []string{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", request.Method)
		}
		if request.URL.Path != "/dashboard/api/rgw/user" {
			t.Errorf("path = %q, want /dashboard/api/rgw/user", request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) != 0 {
			t.Errorf("body = %q, want empty", body)
		}
		wantQuery := url.Values{
			"uid":               {"alice"},
			"display_name":      {"Alice Example"},
			"email":             {""},
			"max_buckets":       {"-1"},
			"system":            {"false"},
			"suspended":         {"true"},
			"generate_key":      {"false"},
			"access_key":        {"access"},
			"secret_key":        {"secret"},
			"daemon_name":       {"rgw.one"},
			"account_id":        {"RGW123"},
			"account_root_user": {"false"},
			"account_policies":  {`{"attach":["arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"],"detach":[]}`},
		}
		if got := request.URL.Query(); !reflect.DeepEqual(got, wantQuery) {
			t.Errorf("query = %#v, want %#v", got, wantQuery)
		}

		writer.Header().Set("Content-Type", defaultMediaType)
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"uid":"alice","full_user_id":"alice","user_id":"alice","display_name":"Alice Example"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	user, err := client.CreateUser(context.Background(), CreateUserRequest{
		UID:             "alice",
		DisplayName:     "Alice Example",
		Email:           &email,
		MaxBuckets:      &maxBuckets,
		System:          &system,
		Suspended:       &suspended,
		GenerateKey:     &generateKey,
		AccessKey:       &accessKey,
		SecretKey:       &secretKey,
		DaemonName:      "rgw.one",
		AccountID:       &accountID,
		AccountRootUser: &accountRoot,
		AccountPolicies: &policies,
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.UID != "alice" || user.DisplayName != "Alice Example" {
		t.Errorf("user = %#v", user)
	}
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	displayName := "Alice Updated"
	email := ""
	maxBuckets := int64(0)
	system := true
	suspended := false

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", request.Method)
		}
		if got := request.URL.EscapedPath(); got != "/dashboard/api/rgw/user/tenant$alice%2Fadmin" {
			t.Errorf("escaped path = %q, want encoded user path segment", got)
		}
		wantQuery := url.Values{
			"display_name": {"Alice Updated"},
			"email":        {""},
			"max_buckets":  {"0"},
			"system":       {"true"},
			"suspended":    {"false"},
		}
		if got := request.URL.Query(); !reflect.DeepEqual(got, wantQuery) {
			t.Errorf("query = %#v, want %#v", got, wantQuery)
		}

		writer.Header().Set("Content-Type", defaultMediaType)
		_, _ = writer.Write([]byte(`{"uid":"tenant$alice/admin","full_user_id":"tenant$alice/admin","display_name":"Alice Updated","email":"","max_buckets":0,"system":true,"suspended":0}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	user, err := client.UpdateUser(context.Background(), UpdateUserRequest{
		UID:         "tenant$alice/admin",
		DisplayName: &displayName,
		Email:       &email,
		MaxBuckets:  &maxBuckets,
		System:      &system,
		Suspended:   &suspended,
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.DisplayName != "Alice Updated" || user.MaxBuckets != 0 || !user.System {
		t.Errorf("user = %#v", user)
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", request.Method)
		}
		if got := request.URL.EscapedPath(); got != "/dashboard/api/rgw/user/tenant$alice%2Fadmin" {
			t.Errorf("escaped path = %q, want encoded user path segment", got)
		}
		if got := request.URL.Query(); !reflect.DeepEqual(got, url.Values{"daemon_name": {"rgw.one"}}) {
			t.Errorf("query = %#v, want daemon_name=rgw.one", got)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteUser(context.Background(), DeleteUserRequest{UID: "tenant$alice/admin", DaemonName: "rgw.one"}); err != nil {
		t.Fatal(err)
	}
}

func TestUserValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient("https://ceph.example")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{name: "list nil context", call: func() error { _, err := client.ListUsers(nil, ListUsersRequest{}); return err }},
		{name: "get nil context", call: func() error { _, err := client.GetUser(nil, GetUserRequest{UID: "alice"}); return err }},
		{name: "get empty UID", call: func() error { _, err := client.GetUser(context.Background(), GetUserRequest{}); return err }},
		{name: "create nil context", call: func() error {
			_, err := client.CreateUser(nil, CreateUserRequest{UID: "alice", DisplayName: "Alice"})
			return err
		}},
		{name: "create empty UID", call: func() error {
			_, err := client.CreateUser(context.Background(), CreateUserRequest{DisplayName: "Alice"})
			return err
		}},
		{name: "create empty display name", call: func() error {
			_, err := client.CreateUser(context.Background(), CreateUserRequest{UID: "alice"})
			return err
		}},
		{name: "update nil context", call: func() error { _, err := client.UpdateUser(nil, UpdateUserRequest{UID: "alice"}); return err }},
		{name: "update empty UID", call: func() error { _, err := client.UpdateUser(context.Background(), UpdateUserRequest{}); return err }},
		{name: "delete nil context", call: func() error { return client.DeleteUser(nil, DeleteUserRequest{UID: "alice"}) }},
		{name: "delete empty UID", call: func() error { return client.DeleteUser(context.Background(), DeleteUserRequest{}) }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.call(); err == nil {
				t.Fatal("error = nil, want validation error")
			}
		})
	}
}
