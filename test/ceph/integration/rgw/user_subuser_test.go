package integration

import (
	"errors"
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
		KeyType: rgw.UserKeyTypeSwift, SecretKey: "initial-subuser-secret",
	}); err != nil {
		t.Fatal(err)
	}
	_, err := client.UpdateSubuser(ctx, rgw.UpdateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", SecretKey: "rotated-subuser-secret",
	})
	var apiErr *rgw.APIError
	if err == nil || errors.As(err, &apiErr) {
		t.Fatalf("omitted Access error = %v, want local validation error", err)
	}
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Subusers) != 1 || user.Subusers[0].Permissions != rgw.SubuserPermissionRead ||
		len(user.SwiftKeys) != 1 || user.SwiftKeys[0].SecretKey != "initial-subuser-secret" {
		t.Fatal("omitted Access changed subuser permissions or credentials")
	}
	if _, err := client.UpdateSubuser(ctx, rgw.UpdateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: new(rgw.SubuserAccessRead),
		KeyType: rgw.UserKeyTypeSwift, SecretKey: "rotated-subuser-secret",
	}); err != nil {
		t.Fatal(err)
	}
	user, err = client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Subusers) != 1 || user.Subusers[0].Permissions != rgw.SubuserPermissionRead ||
		len(user.SwiftKeys) != 1 || user.SwiftKeys[0].SecretKey != "rotated-subuser-secret" {
		t.Fatal("explicit Access did not retain permissions while rotating the secret")
	}
	subusers, err := client.UpdateSubuser(ctx, rgw.UpdateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: new(rgw.SubuserAccessFull),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(subusers) != 1 || subusers[0].Permissions != rgw.SubuserPermissionFull {
		t.Fatalf("updated Admin Ops subusers = %#v", subusers)
	}
	subusers, err = client.UpdateSubuser(ctx, rgw.UpdateSubuserRequest{
		UID: fixture.uid, Subuser: "swift", Access: new(rgw.SubuserAccessNone),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(subusers) != 1 || subusers[0].Permissions != rgw.SubuserPermissionNone {
		t.Fatalf("cleared Admin Ops subusers = %#v", subusers)
	}
	user, err = client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Subusers) != 1 || user.Subusers[0].Permissions != rgw.SubuserPermissionNone ||
		len(user.SwiftKeys) != 1 || user.SwiftKeys[0].SecretKey != "rotated-subuser-secret" {
		t.Fatal("clearing permissions did not persist or changed credentials")
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
