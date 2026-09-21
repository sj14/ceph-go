package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWSetBucketSync(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	if err := client.SetBucketSync(ctx, rgw.SetBucketSyncRequest{
		Name: fixture.name, Enabled: new(false),
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.SetBucketSync(ctx, rgw.SetBucketSyncRequest{
		Name: fixture.name, Enabled: new(true),
	}); err != nil {
		t.Fatal(err)
	}
}
