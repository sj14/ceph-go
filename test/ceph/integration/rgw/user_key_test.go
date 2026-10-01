package integration

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWCreateS3Key(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	accessKey := uniqueResourceName(t, "RGWGOADMINKEY")
	keys, err := client.CreateS3Key(ctx, rgw.CreateS3KeyRequest{
		UID:         fixture.uid,
		AccessKey:   accessKey,
		SecretKey:   "ceph-go-admin-integration-secret",
		GenerateKey: new(false),
		Active:      new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].AccessKey != accessKey ||
		keys[0].SecretKey != "ceph-go-admin-integration-secret" || !keys[0].Active {
		t.Fatalf("created Admin Ops keys = %#v", keys)
	}
	first := keys[0]
	keys, err = client.CreateS3Key(ctx, rgw.CreateS3KeyRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("S3 key collection after second creation = %#v", keys)
	}
	foundFirst := false
	for _, key := range keys {
		if key == first {
			foundFirst = true
		} else if key.User != fixture.uid || key.AccessKey == "" || key.SecretKey == "" || !key.Active {
			t.Fatalf("generated S3 key = %#v", key)
		}
	}
	if !foundFirst {
		t.Fatalf("creation response omitted the existing S3 key: %#v", keys)
	}
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(keys, user.Keys) {
		t.Fatalf("creation response = %#v, stored S3 keys = %#v", keys, user.Keys)
	}
}

// Ceph v20.2.4's RGWUserAdminOp_Key::create returns all keys of the chosen
// type, even when the new Swift credential belongs to a different subuser.
func TestRGWCreateSwiftKey(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	var keys []rgw.SwiftKey
	var first rgw.SwiftKey
	for i, subuser := range []string{"swift-one", "swift-two"} {
		if _, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
			UID: fixture.uid, Subuser: subuser, Access: rgw.SubuserAccessReadWrite,
		}); err != nil {
			t.Fatal(err)
		}
		request := rgw.CreateSwiftKeyRequest{UID: fixture.uid, Subuser: subuser}
		if i == 0 {
			request.GenerateKey = new(false)
			request.SecretKey = "ceph-go-admin-swift-secret"
		}
		var err error
		keys, err = client.CreateSwiftKey(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != i+1 {
			t.Fatalf("Swift key collection after creation %d = %#v", i+1, keys)
		}
		foundNew := false
		for _, key := range keys {
			if key.User == fixture.uid+":"+subuser {
				foundNew = true
				if key.SecretKey == "" || !key.Active || (i == 0 && key.SecretKey != request.SecretKey) {
					t.Fatalf("created Swift key = %#v", key)
				}
				if i == 0 {
					first = key
				}
			} else if key != first {
				t.Fatalf("existing Swift key changed: %#v, want %#v", key, first)
			}
		}
		if !foundNew {
			t.Fatalf("creation response omitted new Swift key: %#v", keys)
		}
	}
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(keys, user.SwiftKeys) {
		t.Fatalf("creation response = %#v, stored Swift keys = %#v", keys, user.SwiftKeys)
	}
	if err := client.DeleteKey(ctx, rgw.DeleteKeyRequest{
		UID: fixture.uid, Subuser: "swift-two", KeyType: rgw.UserKeyTypeSwift,
	}); err != nil {
		t.Fatal(err)
	}
	user, err = client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.SwiftKeys) != 1 || user.SwiftKeys[0] != first {
		t.Fatalf("Swift keys after deletion = %#v", user.SwiftKeys)
	}
}

func TestRGWDeleteKey(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	accessKey := uniqueResourceName(t, "RGWGOADMINKEY")
	if _, err := client.CreateS3Key(ctx, rgw.CreateS3KeyRequest{
		UID: fixture.uid, AccessKey: accessKey,
		SecretKey: "ceph-go-admin-integration-secret", GenerateKey: new(false),
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteKey(ctx, rgw.DeleteKeyRequest{
		UID: fixture.uid, KeyType: rgw.UserKeyTypeS3, AccessKey: accessKey,
	}); err != nil {
		t.Fatal(err)
	}
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Keys) != 0 {
		t.Fatalf("keys after Admin Ops delete = %#v", user.Keys)
	}
}

func TestRGWCreateS3KeyConflict(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	owner, _ := createRGWUserFixture(t, client, ctx)
	other, _ := createRGWUserFixture(t, client, ctx)
	request := rgw.CreateS3KeyRequest{
		UID:       owner.uid,
		AccessKey: uniqueResourceName(t, "RGWGOADMINKEY"),
		SecretKey: "ceph-go-admin-integration-secret", GenerateKey: new(false),
	}
	if _, err := client.CreateS3Key(ctx, request); err != nil {
		t.Fatal(err)
	}
	request.UID = other.uid
	_, err := client.CreateS3Key(ctx, request)
	if !errors.Is(err, rgw.ErrKeyExists) {
		t.Fatalf("duplicate access key error = %v, want KeyExists", err)
	}
	requireRGWAPIStatus(t, err, http.StatusConflict)
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: other.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Keys) != 0 {
		t.Fatalf("rejected duplicate key changed user keys: %#v", user.Keys)
	}
}
