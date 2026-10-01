package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetUserQuotas(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	quotas, err := client.GetUserQuotas(ctx, rgw.GetUserQuotaRequest{UID: fixture.uid})
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
		MaxSize:    new(int64(128*1024 + 1)),
		MaxObjects: new(int64(17)),
	}); err != nil {
		t.Fatal(err)
	}
	quota, err := client.GetUserQuota(ctx, rgw.GetUserQuotaRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if !quota.Enabled || quota.MaxSize != 128*1024+1 || quota.MaxObjects != 17 {
		t.Fatalf("updated Admin Ops user quota = %#v", quota)
	}
	if err := client.SetUserQuota(ctx, rgw.SetUserQuotaRequest{
		UID: fixture.uid, Scope: rgw.QuotaScopeUser, Enabled: new(false),
	}); err != nil {
		t.Fatal(err)
	}
	quota, err = client.GetUserQuota(ctx, rgw.GetUserQuotaRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if quota.Enabled || quota.MaxSize != 128*1024+1 || quota.MaxObjects != 17 {
		t.Fatalf("partially updated Admin Ops user quota = %#v", quota)
	}
}

func TestRGWGetUserBucketQuota(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if err := client.SetUserQuota(ctx, rgw.SetUserQuotaRequest{
		UID: fixture.uid, Scope: rgw.QuotaScopeBucket,
		Enabled: new(true), MaxSize: new(int64(4096)), MaxObjects: new(int64(9)),
	}); err != nil {
		t.Fatal(err)
	}
	quota, err := client.GetUserBucketQuota(ctx, rgw.GetUserQuotaRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if !quota.Enabled || quota.MaxSize != 4096 || quota.MaxObjects != 9 {
		t.Fatalf("user per-bucket quota = %#v", quota)
	}
	quotas, err := client.GetUserQuotas(ctx, rgw.GetUserQuotaRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if quotas.Bucket != quota || quotas.User.Enabled || quotas.User.MaxSize != -1 || quotas.User.MaxObjects != -1 {
		t.Fatalf("both quotas after per-bucket update = %#v", quotas)
	}
}
