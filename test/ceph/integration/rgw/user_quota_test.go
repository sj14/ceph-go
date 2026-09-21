package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetUserQuota(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	quotas, err := client.GetUserQuota(ctx, rgw.GetUserQuotaRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if quotas.User.MaxSize != -1 || quotas.User.MaxObjects != -1 ||
		quotas.Bucket.MaxSize != -1 || quotas.Bucket.MaxObjects != -1 {
		t.Fatalf("Admin Ops user quotas = %#v", quotas)
	}
}

func TestRGWSetUserQuota(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if err := client.SetUserQuota(ctx, rgw.SetUserQuotaRequest{
		UID:        fixture.uid,
		Scope:      rgw.QuotaScopeUser,
		Enabled:    new(true),
		MaxSizeKB:  new(int64(128)),
		MaxObjects: new(int64(17)),
	}); err != nil {
		t.Fatal(err)
	}
	quotas, err := client.GetUserQuota(ctx, rgw.GetUserQuotaRequest{
		UID:   fixture.uid,
		Scope: rgw.QuotaScopeUser,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !quotas.User.Enabled || quotas.User.MaxSize != 128*1024 || quotas.User.MaxObjects != 17 {
		t.Fatalf("updated Admin Ops user quota = %#v", quotas.User)
	}
}
