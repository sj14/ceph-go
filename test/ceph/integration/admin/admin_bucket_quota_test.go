package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminSetBucketQuota(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	if err := client.SetBucketQuota(ctx, admin.SetBucketQuotaRequest{
		UID:        "rgw-go-admin",
		Name:       fixture.name,
		Enabled:    new(true),
		MaxSizeKB:  new(int64(64)),
		MaxObjects: new(int64(9)),
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := client.GetBucket(ctx, admin.GetBucketRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if !bucket.Quota.Enabled || bucket.Quota.MaxSize != 64*1024 || bucket.Quota.MaxObjects != 9 {
		t.Fatalf("updated Admin Ops bucket quota = %#v", bucket.Quota)
	}
}
