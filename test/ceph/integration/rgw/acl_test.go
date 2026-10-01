package integration

import (
	"errors"
	"net/http"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetBucketACL(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	policy, err := client.GetBucketACL(ctx, rgw.GetBucketACLRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Owner.ID != "ceph-go-admin" || len(policy.ACL.Users) != 1 ||
		policy.ACL.Users[0].Permission != rgw.ACLPermissionFullControl {
		t.Fatalf("Admin Ops bucket ACL = %#v", policy)
	}
}

func TestRGWGetObjectACL(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	object := "object-with-distinct-acl"
	resource := fixture.name + "/" + object
	requireS3Success(t, ctx, http.MethodPut, resource, []byte("ACL integration test"))
	// Grant public read only on the object so a bucket read cannot pass this check.
	aclXML := []byte(`<AccessControlPolicy xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Owner><ID>ceph-go-admin</ID></Owner><AccessControlList><Grant><Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="CanonicalUser"><ID>ceph-go-admin</ID></Grantee><Permission>FULL_CONTROL</Permission></Grant><Grant><Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="Group"><URI>http://acs.amazonaws.com/groups/global/AllUsers</URI></Grantee><Permission>READ</Permission></Grant></AccessControlList></AccessControlPolicy>`)
	requireS3Success(t, ctx, http.MethodPut, resource+"?acl", aclXML)
	for _, input := range []rgw.GetObjectACLRequest{
		{Bucket: fixture.name}, {Object: object},
	} {
		_, err := client.GetObjectACL(ctx, input)
		var apiErr *rgw.APIError
		if err == nil || errors.As(err, &apiErr) {
			t.Fatalf("incomplete object ACL request error = %v, want local validation error", err)
		}
	}
	policy, err := client.GetObjectACL(ctx, rgw.GetObjectACLRequest{
		Bucket: fixture.name, Object: object,
	})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Owner.ID != "ceph-go-admin" || len(policy.ACL.Users) != 1 ||
		policy.ACL.Users[0].Permission != rgw.ACLPermissionFullControl || len(policy.ACL.Groups) != 1 ||
		policy.ACL.Groups[0].Group != rgw.ACLGroupAllUsers || policy.ACL.Groups[0].Permission != rgw.ACLPermissionRead {
		t.Fatalf("Admin Ops object ACL = %#v", policy)
	}
	bucketPolicy, err := client.GetBucketACL(ctx, rgw.GetBucketACLRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if len(bucketPolicy.ACL.Groups) != 0 {
		t.Fatalf("bucket unexpectedly has object grants: %#v", bucketPolicy)
	}
}
