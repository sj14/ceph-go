package integration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	rgw "github.com/sj14/rgw-go"
)

type bucketFixture struct {
	client  *rgw.Client
	name    string
	deleted bool
}

func createBucketFixture(t *testing.T, client *rgw.Client, ctx context.Context) *bucketFixture {
	t.Helper()
	fixture := &bucketFixture{
		client: client,
		name:   uniqueResourceName(t, "rgw-go-integration-bucket"),
	}
	if err := client.CreateBucket(ctx, rgw.CreateBucketRequest{Name: fixture.name, UID: "rgw-go-test"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if fixture.deleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := fixture.client.DeleteBucket(cleanupCtx, rgw.DeleteBucketRequest{Name: fixture.name}); err != nil {
			t.Errorf("cleanup bucket %q: %v", fixture.name, err)
		}
	})
	return fixture
}

func TestCreateBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != fixture.name || bucket.Owner != "rgw-go-test" {
		t.Fatalf("bucket = %#v", bucket)
	}
}

func TestListBuckets(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	buckets, err := client.ListBuckets(ctx, rgw.ListBucketsRequest{UID: "rgw-go-test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bucket := range buckets {
		if bucket.Name == fixture.name {
			if bucket.Owner != "rgw-go-test" || bucket.ID == "" {
				t.Fatalf("listed bucket = %#v", bucket)
			}
			return
		}
	}
	t.Fatalf("created bucket %q is missing from ListBuckets", fixture.name)
}

func TestGetBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != fixture.name || bucket.Owner != "rgw-go-test" || bucket.ID == "" || bucket.CreationTime == "" {
		t.Fatalf("bucket = %#v", bucket)
	}

	missing := uniqueResourceName(t, "rgw-go-missing-bucket")
	_, err = client.GetBucket(ctx, rgw.GetBucketRequest{Name: missing})
	var apiError *rgw.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusInternalServerError ||
		!strings.Contains(apiError.Body, "NoSuchBucket") {
		t.Fatalf("GetBucket for missing bucket error = %v, want Dashboard HTTP 500 containing NoSuchBucket", err)
	}
}

func TestUpdateBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	bucketFixture := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: bucketFixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateBucket(ctx, rgw.UpdateBucketRequest{
		Name:              bucketFixture.name,
		BucketID:          bucket.ID,
		UID:               "rgw-go-test",
		VersioningState:   rgw.BucketVersioningEnabled,
		EncryptionEnabled: false,
	}); err != nil {
		t.Fatal(err)
	}

	bucket, err = client.GetBucket(ctx, rgw.GetBucketRequest{Name: bucketFixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Owner != "rgw-go-test" || bucket.Versioning != rgw.BucketVersioningEnabled {
		t.Fatalf("updated bucket = %#v", bucket)
	}
}

func TestDeleteBucket(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture := createBucketFixture(t, client, ctx)

	if err := client.DeleteBucket(ctx, rgw.DeleteBucketRequest{Name: fixture.name}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	var apiError *rgw.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusInternalServerError ||
		!strings.Contains(apiError.Body, "NoSuchBucket") {
		t.Fatalf("GetBucket after delete error = %v, want Dashboard HTTP 500 containing NoSuchBucket", err)
	}
}
