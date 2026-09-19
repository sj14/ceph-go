package integration

import (
	"testing"

	rgw "github.com/sj14/rgw-go"
)

func TestAddUserCapability(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	capabilities, err := client.AddUserCapability(ctx, rgw.AddUserCapabilityRequest{
		UID:        fixture.uid,
		Type:       "usage",
		Permission: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCapability(capabilities, "usage", "read") {
		t.Fatalf("created capabilities = %#v", capabilities)
	}
}

func TestDeleteUserCapability(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)
	request := rgw.AddUserCapabilityRequest{
		UID:        fixture.uid,
		Type:       "usage",
		Permission: "read",
	}
	if _, err := client.AddUserCapability(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteUserCapability(ctx, rgw.DeleteUserCapabilityRequest{
		UID:        request.UID,
		Type:       request.Type,
		Permission: request.Permission,
	}); err != nil {
		t.Fatal(err)
	}

	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if hasCapability(user.Capabilities, request.Type, request.Permission) {
		t.Fatalf("capabilities after delete = %#v", user.Capabilities)
	}
}

func hasCapability(capabilities []rgw.UserCapability, capabilityType, permission string) bool {
	for _, capability := range capabilities {
		if capability.Type == capabilityType && capability.Permission == permission {
			return true
		}
	}
	return false
}
