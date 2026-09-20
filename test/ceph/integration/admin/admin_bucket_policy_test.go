package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminGetBucketPolicy(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	policy, err := client.GetBucketPolicy(ctx, admin.GetBucketPolicyRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Owner.ID != "rgw-go-admin" || len(policy.ACL.Users) != 1 ||
		policy.ACL.Users[0].Permission != admin.ACLPermissionFullControl {
		t.Fatalf("Admin Ops bucket policy = %#v", policy)
	}
}
