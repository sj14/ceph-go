package integration

import (
	"errors"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWSetBucketSync(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	checkSyncStopped := func(want bool) {
		t.Helper()
		info, err := client.GetBucketIndexLogInfo(ctx, rgw.GetBucketIndexLogInfoRequest{
			Bucket: fixture.name, BucketInstance: fixture.name + ":" + bucket.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		if info.SyncStopped != want {
			t.Fatalf("bucket-index SyncStopped = %t, want %t", info.SyncStopped, want)
		}
	}
	if err := client.SetBucketSync(ctx, rgw.SetBucketSyncRequest{
		Name: fixture.name, Sync: new(false),
	}); err != nil {
		t.Fatal(err)
	}
	checkSyncStopped(true)
	err = client.SetBucketSync(ctx, rgw.SetBucketSyncRequest{Name: fixture.name})
	var apiErr *rgw.APIError
	if err == nil || errors.As(err, &apiErr) {
		t.Fatalf("omitted Sync error = %v, want local validation error", err)
	}
	checkSyncStopped(true)
	if err := client.SetBucketSync(ctx, rgw.SetBucketSyncRequest{
		Name: fixture.name, Sync: new(true),
	}); err != nil {
		t.Fatal(err)
	}
	checkSyncStopped(false)
}
