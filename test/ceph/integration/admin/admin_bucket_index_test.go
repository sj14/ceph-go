package integration

import (
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminCheckBucketIndex(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createAdminBucketFixture(t, client, ctx)
	result, err := client.CheckBucketIndex(ctx, admin.CheckBucketIndexRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if result.InvalidMultipartEntries == nil || result.Result.Existing.Usage == nil ||
		result.Result.Calculated.Usage == nil {
		t.Fatalf("Admin Ops bucket index check = %#v", result)
	}
}
