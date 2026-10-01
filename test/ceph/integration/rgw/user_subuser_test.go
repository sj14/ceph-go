package integration

import (
	"errors"
	"reflect"
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

func TestRGWRejectQualifiedSubusers(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	var users []rgw.User
	for range 2 {
		fixture, _ := createRGWUserFixture(t, client, ctx)
		if _, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
			UID: fixture.uid, Subuser: "reader", Access: rgw.SubuserAccessRead,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := client.CreateS3Key(ctx, rgw.CreateS3KeyRequest{
			UID: fixture.uid, Subuser: "reader",
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := client.CreateSwiftKey(ctx, rgw.CreateSwiftKeyRequest{
			UID: fixture.uid, Subuser: "reader", SecretKey: "original-swift-secret", GenerateKey: new(false),
		}); err != nil {
			t.Fatal(err)
		}
		user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	uid := users[0].ID
	operations := []struct {
		name string
		call func(string) error
	}{
		{"CreateSubuser", func(prefix string) error {
			_, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
				UID: uid, Subuser: prefix + ":new-reader", Access: rgw.SubuserAccessFull,
			})
			return err
		}},
		{"UpdateSubuser", func(prefix string) error {
			_, err := client.UpdateSubuser(ctx, rgw.UpdateSubuserRequest{
				UID: uid, Subuser: prefix + ":reader", Access: new(rgw.SubuserAccessFull),
				KeyType: rgw.UserKeyTypeSwift, SecretKey: "replacement-secret",
			})
			return err
		}},
		{"DeleteSubuser", func(prefix string) error {
			return client.DeleteSubuser(ctx, rgw.DeleteSubuserRequest{
				UID: uid, Subuser: prefix + ":reader", PurgeKeys: new(true),
			})
		}},
		{"CreateS3Key", func(prefix string) error {
			_, err := client.CreateS3Key(ctx, rgw.CreateS3KeyRequest{UID: uid, Subuser: prefix + ":reader"})
			return err
		}},
		{"CreateSwiftKey", func(prefix string) error {
			_, err := client.CreateSwiftKey(ctx, rgw.CreateSwiftKeyRequest{
				UID: uid, Subuser: prefix + ":reader", SecretKey: "replacement-secret", GenerateKey: new(false),
			})
			return err
		}},
		{"DeleteSwiftKey", func(prefix string) error {
			return client.DeleteSwiftKey(ctx, rgw.DeleteSwiftKeyRequest{UID: uid, Subuser: prefix + ":reader"})
		}},
	}
	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			for _, prefix := range []string{uid, users[1].ID} {
				err := operation.call(prefix)
				var apiErr *rgw.APIError
				if err == nil || errors.As(err, &apiErr) {
					t.Fatalf("qualified subuser error = %v, want local validation error", err)
				}
				for _, before := range users {
					after, err := client.GetUser(ctx, rgw.GetUserRequest{UID: before.ID})
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(before, after) {
						t.Fatalf("qualified subuser changed user %q", before.ID)
					}
				}
			}
		})
	}
}
