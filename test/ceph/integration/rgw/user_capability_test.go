package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWAddUserCapabilities(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	capabilities, err := client.AddUserCapabilities(ctx, rgw.AddUserCapabilitiesRequest{
		UID: fixture.uid,
		Capabilities: []rgw.Capability{{
			Type: rgw.CapabilityTypeUsage, Permission: rgw.CapabilityPermissionRead,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(capabilities) != 1 || capabilities[0].Type != rgw.CapabilityTypeUsage ||
		capabilities[0].Permission != rgw.CapabilityPermissionRead {
		t.Fatalf("added Admin Ops capabilities = %#v", capabilities)
	}
}

func TestRGWDeleteUserCapabilities(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	capability := rgw.Capability{
		Type: rgw.CapabilityTypeUsage, Permission: rgw.CapabilityPermissionRead,
	}
	if _, err := client.AddUserCapabilities(ctx, rgw.AddUserCapabilitiesRequest{
		UID: fixture.uid, Capabilities: []rgw.Capability{capability},
	}); err != nil {
		t.Fatal(err)
	}
	capabilities, err := client.DeleteUserCapabilities(ctx, rgw.DeleteUserCapabilitiesRequest{
		UID: fixture.uid, Capabilities: []rgw.Capability{capability},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("capabilities after Admin Ops delete = %#v", capabilities)
	}
}
