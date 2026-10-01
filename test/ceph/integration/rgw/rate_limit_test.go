package integration

import (
	"context"
	"testing"
	"time"

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
	if err := client.SetBucketRateLimit(ctx, rgw.SetBucketRateLimitRequest{
		Bucket: fixture.name,
		RateLimitUpdate: rgw.RateLimitUpdate{
			MaxReadOps: new(int64(31)), MaxWriteOps: new(int64(37)),
			MaxReadBytes: new(int64(2048)), MaxWriteBytes: new(int64(4096)), Enabled: new(true),
		},
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
	if err := client.SetBucketRateLimit(ctx, rgw.SetBucketRateLimitRequest{
		Bucket:          fixture.name,
		RateLimitUpdate: rgw.RateLimitUpdate{MaxReadOps: new(int64(0)), Enabled: new(false)},
	}); err != nil {
		t.Fatal(err)
	}
	limit, err = client.GetBucketRateLimit(ctx, rgw.GetBucketRateLimitRequest{Bucket: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if limit != (rgw.RateLimit{MaxWriteOps: 37, MaxReadBytes: 2048, MaxWriteBytes: 4096}) {
		t.Fatalf("partially updated bucket rate limit = %#v", limit)
	}
}

func TestRGWSetUserRateLimit(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if err := client.SetUserRateLimit(ctx, rgw.SetUserRateLimitRequest{
		UID: fixture.uid,
		RateLimitUpdate: rgw.RateLimitUpdate{
			MaxReadOps:    new(int64(17)),
			MaxWriteOps:   new(int64(23)),
			MaxReadBytes:  new(int64(4096)),
			MaxWriteBytes: new(int64(8192)),
			Enabled:       new(true),
		},
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
	if err := client.SetUserRateLimit(ctx, rgw.SetUserRateLimitRequest{
		UID:             fixture.uid,
		RateLimitUpdate: rgw.RateLimitUpdate{MaxReadOps: new(int64(0)), Enabled: new(false)},
	}); err != nil {
		t.Fatal(err)
	}
	limit, err = client.GetUserRateLimit(ctx, rgw.GetUserRateLimitRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if limit != (rgw.RateLimit{MaxWriteOps: 23, MaxReadBytes: 4096, MaxWriteBytes: 8192}) {
		t.Fatalf("partially updated user rate limit = %#v", limit)
	}
}

func TestRGWSetGlobalRateLimit(t *testing.T) {
	// Global mutations must finish and restore settings before parallel tests run.
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	original, err := client.GetGlobalRateLimits(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []rgw.RateLimitScope{
		rgw.RateLimitScopeUser, rgw.RateLimitScopeBucket, rgw.RateLimitScopeAnonymous,
	} {
		t.Run(string(scope), func(t *testing.T) {
			expected := original
			var target *rgw.RateLimit
			switch scope {
			case rgw.RateLimitScopeUser:
				target = &expected.User
			case rgw.RateLimitScopeBucket:
				target = &expected.Bucket
			case rgw.RateLimitScopeAnonymous:
				target = &expected.Anonymous
			}
			saved := *target
			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := client.SetGlobalRateLimit(cleanupCtx, rgw.SetGlobalRateLimitRequest{
					Scope: scope, RateLimitUpdate: rateLimitUpdate(saved),
				}); err != nil {
					t.Errorf("restore global %s rate limit: %v", scope, err)
				}
			})
			// Keep enforcement disabled while testing stored global defaults.
			*target = rgw.RateLimit{MaxReadOps: 101, MaxWriteOps: 103, MaxReadBytes: 8192, MaxWriteBytes: 16384}
			if err := client.SetGlobalRateLimit(ctx, rgw.SetGlobalRateLimitRequest{
				Scope: scope, RateLimitUpdate: rateLimitUpdate(*target),
			}); err != nil {
				t.Fatal(err)
			}
			got, err := client.GetGlobalRateLimits(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got != expected {
				t.Fatalf("global limits = %#v, want %#v", got, expected)
			}
			if err := client.SetGlobalRateLimit(ctx, rgw.SetGlobalRateLimitRequest{
				Scope: scope, RateLimitUpdate: rgw.RateLimitUpdate{MaxReadOps: new(int64(0))},
			}); err != nil {
				t.Fatal(err)
			}
			target.MaxReadOps = 0
			got, err = client.GetGlobalRateLimits(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got != expected {
				t.Fatalf("partially updated global limits = %#v, want %#v", got, expected)
			}
		})
	}
	got, err := client.GetGlobalRateLimits(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != original {
		t.Fatalf("restored global limits = %#v, want %#v", got, original)
	}
}

func rateLimitUpdate(limit rgw.RateLimit) rgw.RateLimitUpdate {
	return rgw.RateLimitUpdate{
		MaxReadOps: &limit.MaxReadOps, MaxWriteOps: &limit.MaxWriteOps,
		MaxReadBytes: &limit.MaxReadBytes, MaxWriteBytes: &limit.MaxWriteBytes,
		Enabled: &limit.Enabled,
	}
}
