package integration

import (
	"net/http"
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func adminBucketLogFixture(t *testing.T) (*admin.Client, string, string) {
	t.Helper()
	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	bucket, err := client.GetBucket(ctx, admin.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	return client, fixture.name, fixture.name + ":" + bucket.ID
}

func TestAdminGetBucketIndexLogInfo(t *testing.T) {
	t.Parallel()

	client, bucket, instance := adminBucketLogFixture(t)
	info, err := client.GetBucketIndexLogInfo(integrationContext(t), admin.GetBucketIndexLogInfoRequest{
		Bucket: bucket, BucketInstance: instance,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Generations) == 0 || info.Generations[0].NumShards <= 0 {
		t.Fatalf("bucket-index log info = %#v", info)
	}
}

func TestAdminListBucketIndexLogEntries(t *testing.T) {
	t.Parallel()

	client, bucket, instance := adminBucketLogFixture(t)
	entries, err := client.ListBucketIndexLogEntries(integrationContext(t), admin.ListBucketIndexLogEntriesRequest{
		Bucket: bucket, BucketInstance: instance, MaxEntries: new(int64(10)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if entries.Truncated {
		t.Fatalf("bucket-index log entries = %#v", entries)
	}
}

func TestAdminGetBucketIndexLogStatus(t *testing.T) {
	t.Parallel()

	client, bucket, _ := adminBucketLogFixture(t)
	_, err := client.GetBucketIndexLogStatus(integrationContext(t), admin.GetBucketIndexLogStatusRequest{
		Bucket: bucket,
	})
	requireAdminAPIStatus(t, err, http.StatusNotFound)
}
