package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWListMetadataKeys(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	sections, err := client.ListMetadataKeys(ctx, rgw.ListMetadataKeysRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range []rgw.MetadataSection{
		rgw.MetadataSectionAccount, rgw.MetadataSectionBucket,
		rgw.MetadataSectionBucketInstance, rgw.MetadataSectionGroup,
		rgw.MetadataSectionOTP, rgw.MetadataSectionRoles,
		rgw.MetadataSectionTopic, rgw.MetadataSectionUser,
	} {
		if !containsString(sections.Keys, string(section)) {
			t.Errorf("metadata sections missing %q: %#v", section, sections)
		}
	}

	users, err := client.ListMetadataKeys(ctx, rgw.ListMetadataKeysRequest{
		Section: rgw.MetadataSectionUser, MaxEntries: new(int64(100)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if users.Count == 0 || users.Truncated || !containsString(users.Keys, "ceph-go-admin") {
		t.Fatalf("user metadata keys = %#v", users)
	}
}

func TestRGWGetMetadata(t *testing.T) {
	t.Parallel()

	metadata, err := rgwIntegrationClient(t).GetMetadata(integrationContext(t), rgw.GetMetadataRequest{
		Section: rgw.MetadataSectionUser, Key: "ceph-go-admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Key != "user:ceph-go-admin" || metadata.Version.Version == 0 ||
		metadata.ModificationTime == nil || metadata.Data["user_id"] != "ceph-go-admin" {
		t.Fatalf("Admin Ops metadata = %#v", metadata)
	}
}

func TestRGWGetMetadataMyself(t *testing.T) {
	t.Parallel()

	metadata, err := rgwIntegrationClient(t).GetMetadataMyself(integrationContext(t), rgw.GetMetadataMyselfRequest{
		Section: rgw.MetadataSectionUser,
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Key != "user:ceph-go-admin" || metadata.Data["user_id"] != "ceph-go-admin" {
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
