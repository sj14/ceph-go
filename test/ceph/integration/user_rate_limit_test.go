package integration

import (
	"testing"

	rgw "github.com/sj14/rgw-go"
)

func TestGetGlobalUserRateLimit(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	configuration, err := client.GetGlobalUserRateLimit(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Bucket.Enabled || configuration.User.Enabled || configuration.Anonymous.Enabled {
		t.Fatalf("global rate limits = %#v, want disabled defaults", configuration)
	}
}

func TestGetUserRateLimit(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	configuration, err := client.GetUserRateLimit(ctx, rgw.GetUserRateLimitRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if configuration.User != (rgw.RateLimit{}) {
		t.Fatalf("initial user rate limit = %#v, want zero value", configuration.User)
	}
}

func TestUpdateUserRateLimit(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)
	want := rgw.RateLimit{
		Enabled:       true,
		MaxReadOps:    201,
		MaxWriteOps:   202,
		MaxReadBytes:  203,
		MaxWriteBytes: 204,
	}
	if err := client.UpdateUserRateLimit(ctx, rgw.UpdateUserRateLimitRequest{
		UID:       fixture.uid,
		RateLimit: want,
	}); err != nil {
		t.Fatal(err)
	}

	configuration, err := client.GetUserRateLimit(ctx, rgw.GetUserRateLimitRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if configuration.User != want {
		t.Fatalf("updated user rate limit = %#v, want %#v", configuration.User, want)
	}
}
