package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminSetBucketSync(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	if err := client.SetBucketSync(ctx, admin.SetBucketSyncRequest{
		Name: fixture.name, Enabled: new(false),
	}); err != nil {
		t.Fatal(err)
	}
	if err := client.SetBucketSync(ctx, admin.SetBucketSyncRequest{
		Name: fixture.name, Enabled: new(true),
	}); err != nil {
		t.Fatal(err)
	}
}
