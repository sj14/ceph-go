package integration

import (
	"testing"

	mgr "github.com/sj14/rgw-go/mgr"
)

func TestGetUserQuota(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	quota, err := client.GetUserQuota(ctx, mgr.GetUserQuotaRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if quota.User.Enabled || quota.Bucket.Enabled {
		t.Fatalf("initial quota = %#v, want both scopes disabled", quota)
	}
}

func TestUpdateUserQuota(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	if err := client.UpdateUserQuota(ctx, mgr.UpdateUserQuotaRequest{
		UID:        fixture.uid,
		Type:       mgr.UserQuotaTypeUser,
		Enabled:    true,
		MaxSizeKB:  2048,
		MaxObjects: 101,
	}); err != nil {
		t.Fatal(err)
	}

	quota, err := client.GetUserQuota(ctx, mgr.GetUserQuotaRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if !quota.User.Enabled || quota.User.MaxSizeKB != 2048 || quota.User.MaxObjects != 101 {
		t.Fatalf("updated user quota = %#v", quota.User)
	}
}
