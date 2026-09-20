package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sj14/rgw-go/admin"
)

type adminBucketFixture struct {
	client  *admin.Client
	name    string
	deleted bool
}

func createAdminBucketFixture(t *testing.T, adminClient *admin.Client, ctx context.Context) *adminBucketFixture {
	t.Helper()
	name := uniqueResourceName(t, "rgw-go-admin-integration-bucket")
	requireS3Success(t, ctx, http.MethodPut, name, nil)
	fixture := &adminBucketFixture{client: adminClient, name: name}
	t.Cleanup(func() {
		if fixture.deleted {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := fixture.client.DeleteBucket(cleanupCtx, admin.DeleteBucketRequest{
			Name:         fixture.name,
			PurgeObjects: new(true),
		}); err != nil {
			t.Errorf("cleanup Admin Ops bucket %q: %v", fixture.name, err)
		}
	})
	return fixture
}

func TestAdminListBuckets(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	buckets, err := client.ListBuckets(ctx, admin.ListBucketsRequest{
		UID:   "rgw-go-admin",
		Stats: new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.Name == fixture.name {
			if bucket.Owner != "rgw-go-admin" || bucket.ID == "" || bucket.IndexType != admin.BucketIndexNormal {
				t.Fatalf("listed Admin Ops bucket = %#v", bucket)
			}
			return
		}
	}
	t.Fatalf("created bucket %q is missing from Admin Ops ListBuckets: %#v", fixture.name, buckets)
}

func TestAdminGetBucket(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	bucket, err := client.GetBucket(ctx, admin.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != fixture.name || bucket.Owner != "rgw-go-admin" || bucket.ID == "" ||
		bucket.Versioning != admin.BucketVersioningOff || bucket.CreationTime.IsZero() {
		t.Fatalf("Admin Ops bucket = %#v", bucket)
	}
}

func TestAdminLinkBucket(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	userFixture, _ := createAdminUserFixture(t, client, ctx)
	bucketFixture := createAdminBucketFixture(t, client, ctx)
	if err := client.LinkBucket(ctx, admin.LinkBucketRequest{
		Name: bucketFixture.name,
		UID:  userFixture.uid,
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := client.GetBucket(ctx, admin.GetBucketRequest{Name: bucketFixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Owner != userFixture.uid {
		t.Fatalf("linked Admin Ops bucket owner = %q, want %q", bucket.Owner, userFixture.uid)
	}
}

func TestAdminUnlinkBucket(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	if err := client.UnlinkBucket(ctx, admin.UnlinkBucketRequest{
		Name: fixture.name,
		UID:  "rgw-go-admin",
	}); err != nil {
		t.Fatal(err)
	}
	buckets, err := client.ListBuckets(ctx, admin.ListBucketsRequest{UID: "rgw-go-admin"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.Name == fixture.name {
			t.Fatalf("unlinked bucket %q remains in the owner's bucket list", fixture.name)
		}
	}
}

func TestAdminDeleteBucket(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	if err := client.DeleteBucket(ctx, admin.DeleteBucketRequest{Name: fixture.name}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetBucket(ctx, admin.GetBucketRequest{Name: fixture.name})
	if !errors.Is(err, admin.ErrNoSuchBucket) {
		t.Fatalf("GetBucket after Admin Ops delete error = %v, want NoSuchBucket", err)
	}
}
