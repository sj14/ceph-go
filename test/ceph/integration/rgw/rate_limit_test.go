package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetRateLimit(t *testing.T) {
	t.Parallel()

	configuration, err := rgwIntegrationClient(t).GetRateLimit(
		integrationContext(t), rgw.GetRateLimitRequest{Global: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Bucket == nil || configuration.User == nil || configuration.Anonymous == nil {
		t.Fatalf("global Admin Ops rate limits = %#v", configuration)
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
	configuration, err := client.GetRateLimit(ctx, rgw.GetRateLimitRequest{
		Scope: rgw.RateLimitScopeUser,
		UID:   fixture.uid,
	})
	if err != nil {
		t.Fatal(err)
	}
	if configuration.User == nil || *configuration.User != (rgw.RateLimit{
		MaxReadOps: 17, MaxWriteOps: 23, MaxReadBytes: 4096, MaxWriteBytes: 8192, Enabled: true,
	}) {
		t.Fatalf("updated Admin Ops user rate limit = %#v", configuration.User)
	}
}
