package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminCreateKey(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	accessKey := uniqueResourceName(t, "RGWGOADMINKEY")
	keys, err := client.CreateKey(ctx, admin.CreateKeyRequest{
		UID:         fixture.uid,
		KeyType:     admin.UserKeyTypeS3,
		AccessKey:   accessKey,
		SecretKey:   "rgw-go-admin-integration-secret",
		GenerateKey: new(false),
		Active:      new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys.AccessKeys) != 1 || keys.AccessKeys[0].AccessKey != accessKey ||
		keys.AccessKeys[0].SecretKey != "rgw-go-admin-integration-secret" || !keys.AccessKeys[0].Active {
		t.Fatalf("created Admin Ops keys = %#v", keys)
	}
}

func TestAdminDeleteKey(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	accessKey := uniqueResourceName(t, "RGWGOADMINKEY")
	if _, err := client.CreateKey(ctx, admin.CreateKeyRequest{
		UID: fixture.uid, KeyType: admin.UserKeyTypeS3, AccessKey: accessKey,
		SecretKey: "rgw-go-admin-integration-secret", GenerateKey: new(false),
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteKey(ctx, admin.DeleteKeyRequest{
		UID: fixture.uid, KeyType: admin.UserKeyTypeS3, AccessKey: accessKey,
	}); err != nil {
		t.Fatal(err)
	}
	user, err := client.GetUser(ctx, admin.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Keys) != 0 {
		t.Fatalf("keys after Admin Ops delete = %#v", user.Keys)
	}
}
