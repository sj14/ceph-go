package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sj14/ceph-go/rgw"
)

type rgwBucketFixture struct {
	client  *rgw.Client
	name    string
	deleted bool
}

func createRGWBucketFixture(t *testing.T, rgwClient *rgw.Client, ctx context.Context) *rgwBucketFixture {
	t.Helper()
	name := uniqueResourceName(t, "ceph-go-admin-integration-bucket")
	requireS3Success(t, ctx, http.MethodPut, name, nil)
	fixture := &rgwBucketFixture{client: rgwClient, name: name}
	t.Cleanup(func() {
		if fixture.deleted {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := fixture.client.DeleteBucket(cleanupCtx, rgw.DeleteBucketRequest{
			Name:         fixture.name,
			PurgeObjects: new(true),
		}); err != nil {
			t.Errorf("cleanup Admin Ops bucket %q: %v", fixture.name, err)
		}
	})
	return fixture
}

func TestRGWListBuckets(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	buckets, err := client.ListBuckets(ctx, rgw.ListBucketsRequest{
		UID: "ceph-go-admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.Name == fixture.name {
			if bucket.Owner != "ceph-go-admin" || bucket.ID == "" || bucket.IndexType != rgw.BucketIndexNormal {
				t.Fatalf("listed Admin Ops bucket = %#v", bucket)
			}
			return
		}
	}
	t.Fatalf("created bucket %q is missing from Admin Ops ListBuckets: %#v", fixture.name, buckets)
}

func TestRGWListBucketNames(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	// Check both global and owner-filtered name-only response paths.
	for _, input := range []rgw.ListBucketsRequest{{}, {UID: "ceph-go-admin"}} {
		names, err := client.ListBucketNames(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		if !containsString(names, fixture.name) {
			t.Errorf("created bucket %q missing from name listing for %#v: %#v", fixture.name, input, names)
		}
	}
}

func TestRGWGetBucket(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != fixture.name || bucket.Owner != "ceph-go-admin" || bucket.ID == "" ||
		bucket.Versioning != rgw.BucketVersioningOff || bucket.CreationTime.IsZero() {
		t.Fatalf("Admin Ops bucket = %#v", bucket)
	}
}

func TestRGWLinkBucketToUser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	userFixture, _ := createRGWUserFixture(t, client, ctx)
	bucketFixture := createRGWBucketFixture(t, client, ctx)
	if err := client.LinkBucketToUser(ctx, rgw.LinkBucketToUserRequest{
		Name: bucketFixture.name,
		UID:  userFixture.uid,
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: bucketFixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Owner != userFixture.uid {
		t.Fatalf("linked Admin Ops bucket owner = %q, want %q", bucket.Owner, userFixture.uid)
	}
}

func TestRGWUnlinkBucketFromUser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	if err := client.UnlinkBucketFromUser(ctx, rgw.UnlinkBucketFromUserRequest{
		Name: fixture.name,
		UID:  "ceph-go-admin",
	}); err != nil {
		t.Fatal(err)
	}
	buckets, err := client.ListBucketNames(ctx, rgw.ListBucketsRequest{UID: "ceph-go-admin"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket == fixture.name {
			t.Fatalf("unlinked bucket %q remains in the owner's bucket list", fixture.name)
		}
	}
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Owner != "ceph-go-admin" {
		t.Fatalf("unlink changed bucket owner to %q", bucket.Owner)
	}
	if err := client.LinkBucketToUser(ctx, rgw.LinkBucketToUserRequest{
		Name: fixture.name, UID: "ceph-go-admin", BucketID: bucket.ID,
	}); err != nil {
		t.Fatal(err)
	}
	buckets, err = client.ListBucketNames(ctx, rgw.ListBucketsRequest{UID: "ceph-go-admin"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(buckets, fixture.name) {
		t.Fatal("relinked bucket is missing from the user's bucket list")
	}
}

func rgwAccountBucketFixture(t *testing.T, client *rgw.Client, ctx context.Context) (accountID, userID string, bucket *rgwBucketFixture) {
	t.Helper()
	account, _ := createRGWAccountFixture(t, client, ctx)
	user, _ := createRGWUserFixture(t, client, ctx)
	if _, err := client.UpdateUser(ctx, rgw.UpdateUserRequest{
		UID: user.uid, AccountID: &account.id, DisplayName: &user.uid,
	}); err != nil {
		t.Fatal(err)
	}
	return account.id, user.uid, createRGWBucketFixture(t, client, ctx)
}

func TestRGWLinkBucketToAccount(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	accountID, userID, fixture := rgwAccountBucketFixture(t, client, ctx)
	if err := client.LinkBucketToAccount(ctx, rgw.LinkBucketToAccountRequest{
		Name: fixture.name, AccountID: accountID,
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Owner != accountID {
		t.Fatalf("linked bucket owner = %q, want %q", bucket.Owner, accountID)
	}
	buckets, err := client.ListBucketNames(ctx, rgw.ListBucketsRequest{UID: userID})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(buckets, fixture.name) {
		t.Fatal("linked bucket is missing from the account's bucket list")
	}
}

func TestRGWUnlinkBucketFromAccount(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	accountID, userID, fixture := rgwAccountBucketFixture(t, client, ctx)
	if err := client.LinkBucketToAccount(ctx, rgw.LinkBucketToAccountRequest{
		Name: fixture.name, AccountID: accountID,
	}); err != nil {
		t.Fatal(err)
	}
	before, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.UnlinkBucketFromAccount(ctx, rgw.UnlinkBucketFromAccountRequest{
		Name: fixture.name, AccountID: accountID,
	}); err != nil {
		t.Fatal(err)
	}
	buckets, err := client.ListBucketNames(ctx, rgw.ListBucketsRequest{UID: userID})
	if err != nil {
		t.Fatal(err)
	}
	if containsString(buckets, fixture.name) {
		t.Fatal("unlinked bucket remains in the account's bucket list")
	}
	after, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if after.Owner != accountID || after.ID != before.ID {
		t.Fatalf("unlink changed bucket identity or owner: %#v", after)
	}
	if err := client.LinkBucketToAccount(ctx, rgw.LinkBucketToAccountRequest{
		Name: fixture.name, AccountID: accountID, BucketID: before.ID,
	}); err != nil {
		t.Fatal(err)
	}
	buckets, err = client.ListBucketNames(ctx, rgw.ListBucketsRequest{UID: userID})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(buckets, fixture.name) {
		t.Fatal("relinked bucket is missing from the account's bucket list")
	}
}

func TestRGWDeleteBucket(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	if err := client.DeleteBucket(ctx, rgw.DeleteBucketRequest{Name: fixture.name}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if !errors.Is(err, rgw.ErrNoSuchBucket) {
		t.Fatalf("GetBucket after Admin Ops delete error = %v, want NoSuchBucket", err)
	}
}
