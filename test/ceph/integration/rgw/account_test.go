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
	account, err := client.CreateAccount(ctx, rgw.CreateAccountRequest{
		Name:          uniqueResourceName(t, "ceph-go-admin-integration-account"),
		Email:         uniqueResourceName(t, "account") + "@example.invalid",
		MaxUsers:      new(int64(11)),
		MaxRoles:      new(int64(12)),
		MaxGroups:     new(int64(13)),
		MaxAccessKeys: new(int64(14)),
		MaxBuckets:    new(int64(15)),
	})
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

func TestRGWUpdateAccount(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWAccountFixture(t, client, ctx)
	account, err := client.UpdateAccount(ctx, rgw.UpdateAccountRequest{
		ID: fixture.id, Email: uniqueResourceName(t, "updated-account") + "@example.invalid",
		MaxBuckets: new(int64(0)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != fixture.id || account.Email == "" || account.MaxBuckets != 0 {
		t.Fatalf("updated Admin Ops account = %#v", account)
	}
}

func TestRGWSetAccountQuota(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWAccountFixture(t, client, ctx)
	account, err := client.SetAccountQuota(ctx, rgw.SetAccountQuotaRequest{
		ID: fixture.id, Scope: rgw.QuotaScopeAccount,
		MaxSize: new(int64(8192)), MaxObjects: new(int64(8)), Enabled: new(true),
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
		MaxSize: new(int64(4096)), MaxObjects: new(int64(7)), Enabled: new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != fixture.id || !account.BucketQuota.Enabled ||
		account.BucketQuota.MaxSize != 4096 || account.BucketQuota.MaxObjects != 7 {
		t.Fatalf("account with updated Admin Ops bucket quota = %#v", account)
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
