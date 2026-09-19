package integration

import (
	"testing"

	rgw "github.com/sj14/rgw-go"
)

func TestCreateAccessKey(t *testing.T) {
	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	keys, err := client.CreateAccessKey(ctx, rgw.CreateAccessKeyRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("created key count = %d, want 1", len(keys))
	}
	if keys[0].User != fixture.uid || keys[0].AccessKey == "" || keys[0].SecretKey == "" || !keys[0].Active {
		t.Fatal("created key is missing its user, access key, secret key, or active state")
	}
}

func TestDeleteAccessKey(t *testing.T) {
	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	keys, err := client.CreateAccessKey(ctx, rgw.CreateAccessKeyRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("created key count = %d, want 1", len(keys))
	}
	if err := client.DeleteAccessKey(ctx, rgw.DeleteAccessKeyRequest{
		UID:       fixture.uid,
		AccessKey: keys[0].AccessKey,
	}); err != nil {
		t.Fatal(err)
	}

	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Keys) != 0 {
		t.Fatalf("remaining key count = %d, want 0", len(user.Keys))
	}
}
