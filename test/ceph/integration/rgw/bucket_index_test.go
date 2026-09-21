package integration

import (
	"testing"

	"github.com/sj14/rgw-go/rgw"
)

func TestRGWCheckBucketIndex(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture := createRGWBucketFixture(t, client, ctx)
	result, err := client.CheckBucketIndex(ctx, rgw.CheckBucketIndexRequest{Name: fixture.name})
	if err != nil {
		t.Fatal(err)
	}
	if result.InvalidMultipartEntries == nil || result.Result.Existing.Usage == nil ||
		result.Result.Calculated.Usage == nil {
		t.Fatalf("Admin Ops bucket index check = %#v", result)
	}
}
