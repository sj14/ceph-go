package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminCreateSubuser(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	subusers, err := client.CreateSubuser(ctx, admin.CreateSubuserRequest{
		UID:     fixture.uid,
		Subuser: "swift",
		Access:  admin.SubuserAccessReadWrite,
		KeyType: admin.UserKeyTypeSwift,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(subusers) != 1 || subusers[0].ID != fixture.uid+":swift" ||
		subusers[0].Permissions != admin.SubuserPermissionReadWrite {
		t.Fatalf("created Admin Ops subusers = %#v", subusers)
	}
}

func TestAdminUpdateSubuser(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, admin.CreateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: admin.SubuserAccessRead,
	}); err != nil {
		t.Fatal(err)
	}
	subusers, err := client.UpdateSubuser(ctx, admin.UpdateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: admin.SubuserAccessFull,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(subusers) != 1 || subusers[0].Permissions != admin.SubuserPermissionFull {
		t.Fatalf("updated Admin Ops subusers = %#v", subusers)
	}
}

func TestAdminDeleteSubuser(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, admin.CreateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: admin.SubuserAccessRead,
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteSubuser(ctx, admin.DeleteSubuserRequest{
		UID: fixture.uid, Subuser: "swift",
	}); err != nil {
		t.Fatal(err)
	}
	user, err := client.GetUser(ctx, admin.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Subusers) != 0 {
		t.Fatalf("subusers after Admin Ops delete = %#v", user.Subusers)
	}
}
