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
		UID:   "ceph-go-admin",
		Stats: new(true),
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

func TestRGWLinkBucket(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	userFixture, _ := createRGWUserFixture(t, client, ctx)
	bucketFixture := createRGWBucketFixture(t, client, ctx)
	if err := client.LinkBucket(ctx, rgw.LinkBucketRequest{
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

func TestRGWUnlinkBucket(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	if err := client.UnlinkBucket(ctx, rgw.UnlinkBucketRequest{
		Name: fixture.name,
		UID:  "ceph-go-admin",
	}); err != nil {
		t.Fatal(err)
	}
	buckets, err := client.ListBuckets(ctx, rgw.ListBucketsRequest{UID: "ceph-go-admin"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.Name == fixture.name {
			t.Fatalf("unlinked bucket %q remains in the owner's bucket list", fixture.name)
		}
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
