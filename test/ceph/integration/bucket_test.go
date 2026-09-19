package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	rgw "github.com/sj14/rgw-go"
)

func createBucketFixture(t *testing.T, client *rgw.Client, ctx context.Context) string {
	t.Helper()
	name := fmt.Sprintf("rgw-go-integration-bucket-%d", time.Now().UnixNano())
	if err := client.CreateBucket(ctx, rgw.CreateBucketRequest{Name: name, UID: "rgw-go-test"}); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestCreateBucket(t *testing.T) {
	client := integrationClient(t)
	ctx := integrationContext(t)
	name := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != name || bucket.Owner != "rgw-go-test" {
		t.Fatalf("bucket = %#v", bucket)
	}
}

func TestGetBucket(t *testing.T) {
	client := integrationClient(t)
	ctx := integrationContext(t)
	name := createBucketFixture(t, client, ctx)

	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}
	if bucket.Name != name || bucket.Owner != "rgw-go-test" || bucket.ID == "" || bucket.CreationTime == "" {
		t.Fatalf("bucket = %#v", bucket)
	}

	missing := fmt.Sprintf("rgw-go-missing-bucket-%d", time.Now().UnixNano())
	_, err = client.GetBucket(ctx, rgw.GetBucketRequest{Name: missing})
	var apiError *rgw.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusInternalServerError ||
		!strings.Contains(apiError.Body, "NoSuchBucket") {
		t.Fatalf("GetBucket for missing bucket error = %v, want Dashboard HTTP 500 containing NoSuchBucket", err)
	}
}
