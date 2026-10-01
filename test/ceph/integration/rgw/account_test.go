package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sj14/ceph-go/rgw"
)

type rgwAccountFixture struct {
	client  *rgw.Client
	id      string
	deleted bool
}

func createRGWAccountFixture(t *testing.T, client *rgw.Client, ctx context.Context) (*rgwAccountFixture, rgw.Account) {
	t.Helper()
	return createRGWAccountFixtureFromRequest(t, client, ctx, rgw.CreateAccountRequest{
		Name:          uniqueResourceName(t, "ceph-go-admin-integration-account"),
		Email:         uniqueResourceName(t, "account") + "@example.invalid",
		MaxUsers:      new(int64(11)),
		MaxRoles:      new(int64(12)),
		MaxGroups:     new(int64(13)),
		MaxAccessKeys: new(int64(14)),
		MaxBuckets:    new(int64(15)),
	})
}

func createRGWAccountFixtureFromRequest(t *testing.T, client *rgw.Client, ctx context.Context, input rgw.CreateAccountRequest) (*rgwAccountFixture, rgw.Account) {
	t.Helper()
	account, err := client.CreateAccount(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &rgwAccountFixture{client: client, id: account.ID}
	t.Cleanup(func() {
		if fixture.deleted {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := fixture.client.DeleteAccount(cleanupCtx, rgw.DeleteAccountRequest{ID: fixture.id}); err != nil {
			t.Errorf("cleanup Admin Ops account %q: %v", fixture.id, err)
		}
	})
	return fixture, account
}

func TestRGWCreateAccount(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	fixture, account := createRGWAccountFixture(t, client, integrationContext(t))
	if account.ID != fixture.id || account.Name == "" || account.Email == "" ||
		account.MaxUsers != 11 || account.MaxRoles != 12 || account.MaxGroups != 13 ||
		account.MaxAccessKeys != 14 || account.MaxBuckets != 15 {
		t.Fatalf("created Admin Ops account = %#v", account)
	}
}

func TestRGWGetAccount(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, created := createRGWAccountFixture(t, client, ctx)
	account, err := client.GetAccount(ctx, rgw.GetAccountRequest{ID: fixture.id})
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != fixture.id || account.Name != created.Name || account.Email != created.Email {
		t.Fatalf("Admin Ops account = %#v", account)
	}
}

func TestRGWAccountByName(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	name := uniqueResourceName(t, "account-by-name")
	tenant := uniqueResourceName(t, "tenant")
	var fixtures []*rgwAccountFixture
	var accounts []rgw.Account
	for _, selectedTenant := range []string{"", tenant} {
		fixture, account := createRGWAccountFixtureFromRequest(t, client, ctx, rgw.CreateAccountRequest{
			Name: name, Tenant: selectedTenant,
		})
		fixtures = append(fixtures, fixture)
		accounts = append(accounts, account)
		got, err := client.GetAccountByName(ctx, rgw.GetAccountByNameRequest{Tenant: selectedTenant, Name: name})
		if err != nil {
			t.Fatal(err)
		}
		if got != account {
			t.Fatalf("account by tenant/name = %#v, want %#v", got, account)
		}
	}
	for _, value := range []string{"", " "} {
		_, getIDErr := client.GetAccount(ctx, rgw.GetAccountRequest{ID: value})
		deleteIDErr := client.DeleteAccount(ctx, rgw.DeleteAccountRequest{ID: value})
		_, getNameErr := client.GetAccountByName(ctx, rgw.GetAccountByNameRequest{Tenant: tenant, Name: value})
		deleteNameErr := client.DeleteAccountByName(ctx, rgw.DeleteAccountByNameRequest{Tenant: tenant, Name: value})
		for _, err := range []error{getIDErr, deleteIDErr, getNameErr, deleteNameErr} {
			var apiErr *rgw.APIError
			if err == nil || errors.As(err, &apiErr) {
				t.Fatalf("missing account selector error = %v, want local validation error", err)
			}
		}
	}
	wrongTenant := uniqueResourceName(t, "missing-tenant")
	_, err := client.GetAccountByName(ctx, rgw.GetAccountByNameRequest{Tenant: wrongTenant, Name: name})
	requireRGWAPIStatus(t, err, 404)
	err = client.DeleteAccountByName(ctx, rgw.DeleteAccountByNameRequest{Tenant: wrongTenant, Name: name})
	requireRGWAPIStatus(t, err, 404)
	for _, account := range accounts {
		got, err := client.GetAccount(ctx, rgw.GetAccountRequest{ID: account.ID})
		if err != nil {
			t.Fatal(err)
		}
		if got != account {
			t.Fatalf("rejected deletion changed account: %#v, want %#v", got, account)
		}
	}
	for i := len(accounts) - 1; i >= 0; i-- {
		account := accounts[i]
		if err := client.DeleteAccountByName(ctx, rgw.DeleteAccountByNameRequest{Tenant: account.Tenant, Name: account.Name}); err != nil {
			t.Fatal(err)
		}
		fixtures[i].deleted = true
		_, err := client.GetAccount(ctx, rgw.GetAccountRequest{ID: account.ID})
		requireRGWAPIStatus(t, err, 404)
		_, err = client.GetAccountByName(ctx, rgw.GetAccountByNameRequest{Tenant: account.Tenant, Name: account.Name})
		requireRGWAPIStatus(t, err, 404)
		// Deleting the named tenant must leave the default tenant's account.
		if i > 0 {
			got, err := client.GetAccountByName(ctx, rgw.GetAccountByNameRequest{Name: name})
			if err != nil {
				t.Fatal(err)
			}
			if got != accounts[0] {
				t.Fatalf("deletion changed default tenant's account: %#v", got)
			}
		}
	}
}

func TestRGWUpdateAccount(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, original := createRGWAccountFixture(t, client, ctx)
	otherFixture, other := createRGWAccountFixture(t, client, ctx)
	for _, input := range []rgw.UpdateAccountRequest{
		{Name: other.Name, MaxBuckets: new(int64(0))},
		{Email: other.Email, MaxBuckets: new(int64(0))},
		{ID: " ", Name: other.Name, Email: other.Email, MaxBuckets: new(int64(0))},
	} {
		_, err := client.UpdateAccount(ctx, input)
		var apiErr *rgw.APIError
		if err == nil || errors.As(err, &apiErr) {
			t.Fatalf("missing ID error = %v, want local validation error", err)
		}
	}
	unchanged, err := client.GetAccount(ctx, rgw.GetAccountRequest{ID: otherFixture.id})
	if err != nil {
		t.Fatal(err)
	}
	if unchanged != other {
		t.Fatalf("missing ID changed the account selected by name/email: %#v", unchanged)
	}
	name := uniqueResourceName(t, "updated-account")
	email := name + "@example.invalid"
	account, err := client.UpdateAccount(ctx, rgw.UpdateAccountRequest{
		ID: fixture.id, Name: name, Email: email,
		MaxBuckets: new(int64(0)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != fixture.id || account.Name != name || account.Email != email ||
		account.Tenant != original.Tenant || account.MaxBuckets != 0 ||
		account.MaxUsers != original.MaxUsers {
		t.Fatalf("updated Admin Ops account = %#v", account)
	}
	stored, err := client.GetAccount(ctx, rgw.GetAccountRequest{ID: fixture.id})
	if err != nil {
		t.Fatal(err)
	}
	if stored != account {
		t.Fatalf("stored updated account = %#v, want %#v", stored, account)
	}
	unchanged, err = client.GetAccount(ctx, rgw.GetAccountRequest{ID: otherFixture.id})
	if err != nil {
		t.Fatal(err)
	}
	if unchanged != other {
		t.Fatalf("ID update changed unrelated account: %#v", unchanged)
	}
}

func TestRGWSetAccountQuota(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWAccountFixture(t, client, ctx)
	account, err := client.SetAccountQuota(ctx, rgw.SetAccountQuotaRequest{
		ID: fixture.id, Scope: rgw.QuotaScopeAccount,
		MaxSize: new(int32(8192)), MaxObjects: new(int32(8)), Enabled: new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != fixture.id || !account.Quota.Enabled ||
		account.Quota.MaxSize != 8192 || account.Quota.MaxObjects != 8 {
		t.Fatalf("account with updated Admin Ops quota = %#v", account)
	}

	account, err = client.SetAccountQuota(ctx, rgw.SetAccountQuotaRequest{
		ID: fixture.id, Scope: rgw.QuotaScopeBucket,
		MaxSize: new(int32(4096)), MaxObjects: new(int32(7)), Enabled: new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != fixture.id || !account.BucketQuota.Enabled ||
		account.BucketQuota.MaxSize != 4096 || account.BucketQuota.MaxObjects != 7 {
		t.Fatalf("account with updated Admin Ops bucket quota = %#v", account)
	}
	// Exercise the REST parser's signed 32-bit boundary and unlimited value.
	account, err = client.SetAccountQuota(ctx, rgw.SetAccountQuotaRequest{
		ID: fixture.id, Scope: rgw.QuotaScopeAccount,
		MaxSize: new(int32(2147483647)), MaxObjects: new(int32(-1)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.Quota.MaxSize != 2147483647 || account.Quota.MaxObjects != -1 || !account.Quota.Enabled {
		t.Fatalf("account quota at REST limits = %#v", account.Quota)
	}
	account, err = client.SetAccountQuota(ctx, rgw.SetAccountQuotaRequest{
		ID: fixture.id, Scope: rgw.QuotaScopeAccount,
		MaxObjects: new(int32(0)), Enabled: new(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.Quota.MaxSize != 2147483647 || account.Quota.MaxObjects != 0 || account.Quota.Enabled {
		t.Fatalf("account quota with explicit zero/false and omitted size = %#v", account.Quota)
	}
}

func TestRGWDeleteAccount(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWAccountFixture(t, client, ctx)
	if err := client.DeleteAccount(ctx, rgw.DeleteAccountRequest{ID: fixture.id}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetAccount(ctx, rgw.GetAccountRequest{ID: fixture.id})
	if err == nil {
		t.Fatal("GetAccount after Admin Ops delete error = nil")
	}
	var apiError *rgw.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != 404 {
		t.Fatalf("GetAccount after Admin Ops delete error = %v, want 404 APIError", err)
	}
}
