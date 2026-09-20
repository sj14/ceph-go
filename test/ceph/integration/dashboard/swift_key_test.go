package integration

import (
	"testing"

	rgw "github.com/sj14/rgw-go/dashboard"
)

func TestCreateSwiftKey(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
		UID:            fixture.uid,
		Subuser:        "reader",
		Access:         rgw.SubuserAccessRead,
		KeyType:        rgw.SubuserKeyTypeSwift,
		GenerateSecret: new(false),
	}); err != nil {
		t.Fatal(err)
	}

	keys, err := client.CreateSwiftKey(ctx, rgw.CreateSwiftKeyRequest{
		UID:     fixture.uid,
		Subuser: "reader",
	})
	if err != nil {
		t.Fatal(err)
	}
	key, found := findSwiftKey(keys, fixture.uid+":reader")
	if !found || key.SecretKey == "" || !key.Active {
		t.Fatal("created Swift key is missing its user, secret key, or active state")
	}
}

func TestDeleteSwiftKey(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)
	if _, err := client.CreateSubuser(ctx, rgw.CreateSubuserRequest{
		UID:     fixture.uid,
		Subuser: "writer",
		Access:  rgw.SubuserAccessWrite,
		KeyType: rgw.SubuserKeyTypeSwift,
	}); err != nil {
		t.Fatal(err)
	}

	fullSubuser := fixture.uid + ":writer"
	if err := client.DeleteSwiftKey(ctx, rgw.DeleteSwiftKeyRequest{
		UID:     fixture.uid,
		Subuser: fullSubuser,
	}); err != nil {
		t.Fatal(err)
	}

	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if _, found := findSwiftKey(user.SwiftKeys, fullSubuser); found {
		t.Fatalf("Swift keys after delete = %#v", user.SwiftKeys)
	}
}

func findSwiftKey(keys []rgw.UserSwiftKey, user string) (rgw.UserSwiftKey, bool) {
	for _, key := range keys {
		if key.User == user {
			return key, true
		}
	}
	return rgw.UserSwiftKey{}, false
}
