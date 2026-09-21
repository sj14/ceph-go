package integration

import (
	"testing"

	"github.com/sj14/rgw-go/rgw"
)

func TestRGWListMetadataKeys(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	sections, err := client.ListMetadataKeys(ctx, rgw.ListMetadataKeysRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if sections.Count == 0 || !containsString(sections.Keys, "user") {
		t.Fatalf("metadata sections = %#v", sections)
	}

	users, err := client.ListMetadataKeys(ctx, rgw.ListMetadataKeysRequest{
		Section: "user", MaxEntries: new(int64(100)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if users.Count == 0 || users.Truncated || !containsString(users.Keys, "rgw-go-admin") {
		t.Fatalf("user metadata keys = %#v", users)
	}
}

func TestRGWGetMetadata(t *testing.T) {
	t.Parallel()

	metadata, err := rgwIntegrationClient(t).GetMetadata(integrationContext(t), rgw.GetMetadataRequest{
		Section: "user", Key: "rgw-go-admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Key != "user:rgw-go-admin" || metadata.Version.Version == 0 ||
		metadata.ModificationTime == nil || metadata.Data["user_id"] != "rgw-go-admin" {
		t.Fatalf("Admin Ops metadata = %#v", metadata)
	}
}

func TestRGWGetLocalMetadata(t *testing.T) {
	t.Parallel()

	metadata, err := rgwIntegrationClient(t).GetLocalMetadata(integrationContext(t), rgw.GetLocalMetadataRequest{
		Section: "user",
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Key != "user:rgw-go-admin" || metadata.Data["user_id"] != "rgw-go-admin" {
		t.Fatalf("local Admin Ops metadata = %#v", metadata)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
