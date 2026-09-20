package integration

import (
	"testing"

	rgw "github.com/sj14/rgw-go/dashboard"
)

func TestGetGlobalBucketRateLimit(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	configuration, err := client.GetGlobalBucketRateLimit(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Bucket.Enabled || configuration.User.Enabled || configuration.Anonymous.Enabled {
		t.Fatalf("global rate limits = %#v, want disabled defaults", configuration)
	}
}

func TestGetBucketRateLimit(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	configuration, err := client.GetBucketRateLimit(ctx, rgw.GetBucketRateLimitRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Bucket != (rgw.RateLimit{}) {
		t.Fatalf("initial bucket rate limit = %#v, want zero value", configuration.Bucket)
	}
}

func TestUpdateBucketRateLimit(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)
	want := rgw.RateLimit{
		Enabled:       true,
		MaxReadOps:    101,
		MaxWriteOps:   102,
		MaxReadBytes:  103,
		MaxWriteBytes: 104,
	}
	if err := client.UpdateBucketRateLimit(ctx, rgw.UpdateBucketRateLimitRequest{
		Name:      fixture.name,
		RateLimit: want,
	}); err != nil {
		t.Fatal(err)
	}

	configuration, err := client.GetBucketRateLimit(ctx, rgw.GetBucketRateLimitRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Bucket != want {
		t.Fatalf("updated bucket rate limit = %#v, want %#v", configuration.Bucket, want)
	}
}
