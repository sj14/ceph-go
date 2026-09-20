package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestAccountEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		query  url.Values
		call   func(context.Context, *Client) error
	}{
		{
			name:   "get",
			method: http.MethodGet,
			query:  url.Values{"id": {"RGW00000000000000001"}},
			call: func(ctx context.Context, client *Client) error {
				account, err := client.GetAccount(ctx, GetAccountRequest{ID: "RGW00000000000000001"})
				if err == nil && account.ID != "RGW00000000000000001" {
					return fmt.Errorf("account = %#v", account)
				}
				return err
			},
		},
		{
			name:   "create",
			method: http.MethodPost,
			query: url.Values{
				"id":              {"RGW00000000000000001"},
				"tenant":          {"tenant"},
				"name":            {"name"},
				"email":           {"account@example.invalid"},
				"max-users":       {"1"},
				"max-roles":       {"2"},
				"max-groups":      {"3"},
				"max-access-keys": {"4"},
				"max-buckets":     {"5"},
			},
			call: func(ctx context.Context, client *Client) error {
				_, err := client.CreateAccount(ctx, CreateAccountRequest{
					ID: "RGW00000000000000001", Tenant: "tenant", Name: "name",
					Email: "account@example.invalid", MaxUsers: new(int64(1)),
					MaxRoles: new(int64(2)), MaxGroups: new(int64(3)),
					MaxAccessKeys: new(int64(4)), MaxBuckets: new(int64(5)),
				})
				return err
			},
		},
		{
			name:   "update",
			method: http.MethodPut,
			query: url.Values{
				"id":          {"RGW00000000000000001"},
				"email":       {"updated@example.invalid"},
				"max-buckets": {"0"},
			},
			call: func(ctx context.Context, client *Client) error {
				_, err := client.UpdateAccount(ctx, UpdateAccountRequest{
					ID: "RGW00000000000000001", Email: "updated@example.invalid",
					MaxBuckets: new(int64(0)),
				})
				return err
			},
		},
		{
			name:   "set quota",
			method: http.MethodPut,
			query: url.Values{
				"quota":       {""},
				"id":          {"RGW00000000000000001"},
				"quota-type":  {"bucket"},
				"max-size":    {"1024"},
				"max-objects": {"7"},
				"enabled":     {"true"},
			},
			call: func(ctx context.Context, client *Client) error {
				_, err := client.SetAccountQuota(ctx, SetAccountQuotaRequest{
					ID: "RGW00000000000000001", Scope: QuotaScopeBucket,
					MaxSize: new(int64(1024)), MaxObjects: new(int64(7)), Enabled: new(true),
				})
				return err
			},
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			query:  url.Values{"tenant": {"tenant"}, "name": {"name"}},
			call: func(ctx context.Context, client *Client) error {
				return client.DeleteAccount(ctx, DeleteAccountRequest{Tenant: "tenant", Name: "name"})
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != test.method || request.URL.Path != "/admin/account" {
					t.Errorf("request = %s %s", request.Method, request.URL.Path)
				}
				wantQuery := test.query
				wantQuery.Set("format", "json")
				if !reflect.DeepEqual(request.URL.Query(), wantQuery) {
					t.Errorf("query = %v, want %v", request.URL.Query(), wantQuery)
				}
				if test.method != http.MethodDelete {
					_, _ = writer.Write([]byte(`{"id":"RGW00000000000000001","max_buckets":5}`))
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "access-key", "secret-key")
			if err != nil {
				t.Fatal(err)
			}
			if err := test.call(context.Background(), client); err != nil {
				t.Fatal(err)
			}
		})
	}
}
