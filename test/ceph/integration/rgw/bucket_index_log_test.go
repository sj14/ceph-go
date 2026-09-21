package integration

import (
	"net/http"
	"testing"

	"github.com/sj14/rgw-go/rgw"
)

func rgwBucketLogFixture(t *testing.T) (*rgw.Client, string, string) {
	t.Helper()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	return client, fixture.name, fixture.name + ":" + bucket.ID
}

func TestRGWGetBucketIndexLogInfo(t *testing.T) {
	t.Parallel()

	client, bucket, instance := rgwBucketLogFixture(t)
	info, err := client.GetBucketIndexLogInfo(integrationContext(t), rgw.GetBucketIndexLogInfoRequest{
		Bucket: bucket, BucketInstance: instance,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Generations) == 0 || info.Generations[0].NumShards <= 0 {
		t.Fatalf("bucket-index log info = %#v", info)
	}
}

func TestRGWListBucketIndexLogEntries(t *testing.T) {
	t.Parallel()

	client, bucket, instance := rgwBucketLogFixture(t)
	entries, err := client.ListBucketIndexLogEntries(integrationContext(t), rgw.ListBucketIndexLogEntriesRequest{
		Bucket: bucket, BucketInstance: instance, MaxEntries: new(int64(10)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if entries.Truncated {
		t.Fatalf("bucket-index log entries = %#v", entries)
	}
}

func TestRGWGetBucketIndexLogStatus(t *testing.T) {
	t.Parallel()

	client, bucket, _ := rgwBucketLogFixture(t)
	_, err := client.GetBucketIndexLogStatus(integrationContext(t), rgw.GetBucketIndexLogStatusRequest{
		Bucket: bucket,
	})
	requireRGWAPIStatus(t, err, http.StatusNotFound)
}
