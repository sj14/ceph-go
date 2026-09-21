package integration

import (
	"testing"

	mgr "github.com/sj14/rgw-go/mgr"
)

func TestCreateSubuser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	subusers, err := client.CreateSubuser(ctx, mgr.CreateSubuserRequest{
		UID:     fixture.uid,
		Subuser: "reader",
		Access:  mgr.SubuserAccessRead,
		KeyType: mgr.SubuserKeyTypeSwift,
	})
	if err != nil {
		t.Fatal(err)
	}
	subuser, found := findSubuser(subusers, fixture.uid+":reader")
	if !found || subuser.Permissions != mgr.SubuserPermissionRead {
		t.Fatalf("created subusers = %#v", subusers)
	}
}

func TestDeleteSubuser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, mgr.CreateSubuserRequest{
		UID:     fixture.uid,
		Subuser: "writer",
		Access:  mgr.SubuserAccessWrite,
		KeyType: mgr.SubuserKeyTypeS3,
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteSubuser(ctx, mgr.DeleteSubuserRequest{
		UID:     fixture.uid,
		Subuser: "writer",
	}); err != nil {
		t.Fatal(err)
	}

	user, err := client.GetUser(ctx, mgr.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := findSubuser(user.Subusers, fixture.uid+":writer"); found {
		t.Fatalf("subusers after delete = %#v", user.Subusers)
	}
}

func findSubuser(subusers []mgr.UserSubuser, id string) (mgr.UserSubuser, bool) {
	for _, subuser := range subusers {
		if subuser.ID == id {
			return subuser, true
		}
	}
	return mgr.UserSubuser{}, false
}
