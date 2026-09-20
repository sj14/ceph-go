package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminListUsers(t *testing.T) {
	// Ceph lists user metadata non-atomically, so this test must not overlap
	// parallel tests that delete their user fixtures during cleanup.
	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	users, err := client.ListUsers(ctx, admin.ListUsersRequest{MaxEntries: new(int64(1000))})
	if err != nil {
		t.Fatal(err)
	}
	for _, uid := range users.IDs {
		if uid == fixture.uid {
			return
		}
	}
	t.Fatalf("created user %q is missing from Admin Ops ListUsers: %#v", fixture.uid, users)
}
