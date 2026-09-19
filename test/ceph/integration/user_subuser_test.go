package integration

import (
	"testing"

	rgw "github.com/sj14/rgw-go"
)

func TestCreateSubuser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	subusers, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
		UID:     fixture.uid,
		Subuser: "reader",
		Access:  rgw.SubuserAccessRead,
		KeyType: rgw.SubuserKeyTypeSwift,
	})
	if err != nil {
		t.Fatal(err)
	}
	subuser, found := findSubuser(subusers, fixture.uid+":reader")
	if !found || subuser.Permissions != "read" {
		t.Fatalf("created subusers = %#v", subusers)
	}
}

func TestDeleteSubuser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
		UID:     fixture.uid,
		Subuser: "writer",
		Access:  rgw.SubuserAccessWrite,
		KeyType: rgw.SubuserKeyTypeS3,
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteSubuser(ctx, rgw.DeleteSubuserRequest{
		UID:     fixture.uid,
		Subuser: "writer",
	}); err != nil {
		t.Fatal(err)
	}

	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := findSubuser(user.Subusers, fixture.uid+":writer"); found {
		t.Fatalf("subusers after delete = %#v", user.Subusers)
	}
}

func findSubuser(subusers []rgw.UserSubuser, id string) (rgw.UserSubuser, bool) {
	for _, subuser := range subusers {
		if subuser.ID == id {
			return subuser, true
		}
	}
	return rgw.UserSubuser{}, false
}
