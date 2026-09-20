package integration

import (
	"testing"
	"time"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminGetUsage(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	uid := uniqueResourceName(t, "rgw-go-admin-integration-usage")
	usage, err := client.GetUsage(ctx, admin.GetUsageRequest{
		UID:         uid,
		Start:       new(time.Unix(0, 0)),
		End:         new(time.Now().Add(time.Minute)),
		ShowEntries: new(true),
		ShowSummary: new(true),
		Categories:  []string{"get_obj", "put_obj"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(usage.Entries) != 0 || len(usage.Summary) != 0 {
		t.Fatalf("usage for unknown user %q = %#v", uid, usage)
	}
}

func TestAdminTrimUsage(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	uid := uniqueResourceName(t, "rgw-go-admin-integration-trim-usage")
	if err := client.TrimUsage(ctx, admin.TrimUsageRequest{UID: uid}); err != nil {
		t.Fatal(err)
	}
}
