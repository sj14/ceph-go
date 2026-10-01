package integration

import (
	"testing"
	"time"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetUsage(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	uid := uniqueResourceName(t, "ceph-go-admin-integration-usage")
	usage, err := client.GetUsage(ctx, rgw.GetUsageRequest{
		UID:        uid,
		Start:      new(time.Unix(0, 0)),
		End:        new(time.Now().Add(time.Minute)),
		Categories: []string{"get_obj", "put_obj"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if usage.Entries == nil || usage.Summary == nil || len(usage.Entries) != 0 || len(usage.Summary) != 0 {
		t.Fatalf("usage for unknown user %q = %#v", uid, usage)
	}
}

func TestRGWListUsageEntries(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	uid := uniqueResourceName(t, "ceph-go-admin-integration-usage")
	entries, err := client.ListUsageEntries(integrationContext(t), rgw.GetUsageRequest{
		UID: uid, Start: new(time.Unix(0, 0)), End: new(time.Now().Add(time.Minute)),
		Categories: []string{"get_obj", "put_obj"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if entries == nil || len(entries) != 0 {
		t.Fatalf("usage entries for unknown user %q = %#v, want an empty collection", uid, entries)
	}
}

func TestRGWListUsageSummaries(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	uid := uniqueResourceName(t, "ceph-go-admin-integration-usage")
	summaries, err := client.ListUsageSummaries(integrationContext(t), rgw.GetUsageRequest{
		UID: uid, Start: new(time.Unix(0, 0)), End: new(time.Now().Add(time.Minute)),
		Categories: []string{"get_obj", "put_obj"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if summaries == nil || len(summaries) != 0 {
		t.Fatalf("usage summaries for unknown user %q = %#v, want an empty collection", uid, summaries)
	}
}

func TestRGWTrimUsage(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	uid := uniqueResourceName(t, "ceph-go-admin-integration-trim-usage")
	if err := client.TrimUsage(ctx, rgw.TrimUsageRequest{UID: uid}); err != nil {
		t.Fatal(err)
	}
}
