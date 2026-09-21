package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWCreateSubuser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	subusers, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
		UID:     fixture.uid,
		Subuser: "swift",
		Access:  rgw.SubuserAccessReadWrite,
		KeyType: rgw.UserKeyTypeSwift,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(subusers) != 1 || subusers[0].ID != fixture.uid+":swift" ||
		subusers[0].Permissions != rgw.SubuserPermissionReadWrite {
		t.Fatalf("created Admin Ops subusers = %#v", subusers)
	}
}

func TestRGWUpdateSubuser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: rgw.SubuserAccessRead,
	}); err != nil {
		t.Fatal(err)
	}
	subusers, err := client.UpdateSubuser(ctx, rgw.UpdateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: rgw.SubuserAccessFull,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(subusers) != 1 || subusers[0].Permissions != rgw.SubuserPermissionFull {
		t.Fatalf("updated Admin Ops subusers = %#v", subusers)
	}
}

func TestRGWDeleteSubuser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: rgw.SubuserAccessRead,
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteSubuser(ctx, rgw.DeleteSubuserRequest{
		UID: fixture.uid, Subuser: "swift",
	}); err != nil {
		t.Fatal(err)
	}
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Subusers) != 0 {
		t.Fatalf("subusers after Admin Ops delete = %#v", user.Subusers)
	}
}
