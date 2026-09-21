package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetBucketPolicy(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	policy, err := client.GetBucketPolicy(ctx, rgw.GetBucketPolicyRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Owner.ID != "ceph-go-admin" || len(policy.ACL.Users) != 1 ||
		policy.ACL.Users[0].Permission != rgw.ACLPermissionFullControl {
		t.Fatalf("Admin Ops bucket policy = %#v", policy)
	}
}
