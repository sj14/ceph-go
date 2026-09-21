package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWCreateKey(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	accessKey := uniqueResourceName(t, "RGWGOADMINKEY")
	keys, err := client.CreateKey(ctx, rgw.CreateKeyRequest{
		UID:         fixture.uid,
		KeyType:     rgw.UserKeyTypeS3,
		AccessKey:   accessKey,
		SecretKey:   "ceph-go-admin-integration-secret",
		GenerateKey: new(false),
		Active:      new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys.AccessKeys) != 1 || keys.AccessKeys[0].AccessKey != accessKey ||
		keys.AccessKeys[0].SecretKey != "ceph-go-admin-integration-secret" || !keys.AccessKeys[0].Active {
		t.Fatalf("created Admin Ops keys = %#v", keys)
	}
}

func TestRGWDeleteKey(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	accessKey := uniqueResourceName(t, "RGWGOADMINKEY")
	if _, err := client.CreateKey(ctx, rgw.CreateKeyRequest{
		UID: fixture.uid, KeyType: rgw.UserKeyTypeS3, AccessKey: accessKey,
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
