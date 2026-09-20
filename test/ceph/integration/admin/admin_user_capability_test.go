package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminAddUserCapabilities(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	capabilities, err := client.AddUserCapabilities(ctx, admin.AddUserCapabilitiesRequest{
		UID: fixture.uid,
		Capabilities: []admin.Capability{{
			Type: admin.CapabilityTypeUsage, Permission: admin.CapabilityPermissionRead,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(capabilities) != 1 || capabilities[0].Type != admin.CapabilityTypeUsage ||
		capabilities[0].Permission != admin.CapabilityPermissionRead {
		t.Fatalf("added Admin Ops capabilities = %#v", capabilities)
	}
}

func TestAdminDeleteUserCapabilities(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	capability := admin.Capability{
		Type: admin.CapabilityTypeUsage, Permission: admin.CapabilityPermissionRead,
	}
	if _, err := client.AddUserCapabilities(ctx, admin.AddUserCapabilitiesRequest{
		UID: fixture.uid, Capabilities: []admin.Capability{capability},
	}); err != nil {
		t.Fatal(err)
	}
	capabilities, err := client.DeleteUserCapabilities(ctx, admin.DeleteUserCapabilitiesRequest{
		UID: fixture.uid, Capabilities: []admin.Capability{capability},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("capabilities after Admin Ops delete = %#v", capabilities)
	}
}
