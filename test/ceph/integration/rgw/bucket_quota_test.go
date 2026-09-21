package integration

import (
	"testing"

	"github.com/sj14/rgw-go/rgw"
)

func TestRGWSetBucketQuota(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	if err := client.SetBucketQuota(ctx, rgw.SetBucketQuotaRequest{
		UID:        "rgw-go-admin",
		Name:       fixture.name,
		Enabled:    new(true),
		MaxSizeKB:  new(int64(64)),
		MaxObjects: new(int64(9)),
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if !bucket.Quota.Enabled || bucket.Quota.MaxSize != 64*1024 || bucket.Quota.MaxObjects != 9 {
		t.Fatalf("updated Admin Ops bucket quota = %#v", bucket.Quota)
	}
}
