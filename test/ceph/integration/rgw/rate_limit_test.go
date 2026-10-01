package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetGlobalRateLimits(t *testing.T) {
	t.Parallel()

	configuration, err := rgwIntegrationClient(t).GetGlobalRateLimits(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if configuration != (rgw.GlobalRateLimitConfiguration{}) {
		t.Fatalf("global Admin Ops rate limits = %#v", configuration)
	}
}

func TestRGWGetUserRateLimit(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	limit, err := client.GetUserRateLimit(ctx, rgw.GetUserRateLimitRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if limit != (rgw.RateLimit{}) {
		t.Fatalf("unconfigured user rate limit = %#v", limit)
	}
}

func TestRGWGetBucketRateLimit(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	limit, err := client.GetBucketRateLimit(ctx, rgw.GetBucketRateLimitRequest{Bucket: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if limit != (rgw.RateLimit{}) {
		t.Fatalf("unconfigured bucket rate limit = %#v", limit)
	}
	if err := client.SetRateLimit(ctx, rgw.SetRateLimitRequest{
		Scope: rgw.RateLimitScopeBucket, Bucket: fixture.name,
		MaxReadOps: new(int64(31)), MaxWriteOps: new(int64(37)),
		MaxReadBytes: new(int64(2048)), MaxWriteBytes: new(int64(4096)), Enabled: new(true),
	}); err != nil {
		t.Fatal(err)
	}
	limit, err = client.GetBucketRateLimit(ctx, rgw.GetBucketRateLimitRequest{Bucket: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if limit != (rgw.RateLimit{
		MaxReadOps: 31, MaxWriteOps: 37, MaxReadBytes: 2048, MaxWriteBytes: 4096, Enabled: true,
	}) {
		t.Fatalf("updated bucket rate limit = %#v", limit)
	}
}

func TestRGWSetRateLimit(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if err := client.SetRateLimit(ctx, rgw.SetRateLimitRequest{
		Scope:         rgw.RateLimitScopeUser,
		UID:           fixture.uid,
		MaxReadOps:    new(int64(17)),
		MaxWriteOps:   new(int64(23)),
		MaxReadBytes:  new(int64(4096)),
		MaxWriteBytes: new(int64(8192)),
		Enabled:       new(true),
	}); err != nil {
		t.Fatal(err)
	}
	limit, err := client.GetUserRateLimit(ctx, rgw.GetUserRateLimitRequest{
		UID: fixture.uid,
	})
	if err != nil {
		t.Fatal(err)
	}
	if limit != (rgw.RateLimit{
		MaxReadOps: 17, MaxWriteOps: 23, MaxReadBytes: 4096, MaxWriteBytes: 8192, Enabled: true,
	}) {
		t.Fatalf("updated Admin Ops user rate limit = %#v", limit)
	}
}
