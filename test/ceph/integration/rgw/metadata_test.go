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
		Section: rgw.MetadataSectionUser, MaxEntries: new(int32(100)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if users.Count == 0 || users.Truncated || !containsString(users.Keys, "ceph-go-admin") {
		t.Fatalf("user metadata keys = %#v", users)
	}

	// Ceph accepts these values but returns a non-advancing page or loops.
	for _, maxEntries := range []int32{0, -1} {
		if _, err := client.ListMetadataKeys(ctx, rgw.ListMetadataKeysRequest{MaxEntries: &maxEntries}); err == nil {
			t.Fatalf("ListMetadataKeys with MaxEntries %d succeeded", maxEntries)
		}
	}
}

func TestRGWListMetadataKeysPages(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	userA, _ := createRGWUserFixture(t, client, ctx)
	userB, _ := createRGWUserFixture(t, client, ctx)
	bucketA := createRGWBucketFixture(t, client, ctx)
	bucketB := createRGWBucketFixture(t, client, ctx)
	for section, want := range map[rgw.MetadataSection][]string{
		rgw.MetadataSectionUser:   {userA.uid, userB.uid},
		rgw.MetadataSectionBucket: {bucketA.name, bucketB.name},
	} {
		t.Run(string(section), func(t *testing.T) {
			seen := map[string]bool{}
			input := rgw.ListMetadataKeysRequest{Section: section, MaxEntries: new(int32(1))}
			for pages := 1; ; pages++ {
				page, err := client.ListMetadataKeys(ctx, input)
				if err != nil {
					t.Fatal(err)
				}
				if len(page.Keys) > 1 || page.Count != int64(len(page.Keys)) {
					t.Fatalf("page %d = %#v", pages, page)
				}
				for _, key := range page.Keys {
					if seen[key] {
						t.Fatalf("key %q repeated on page %d", key, pages)
					}
					seen[key] = true
				}
				if !page.Truncated {
					if pages < 2 {
						t.Fatalf("listing ended after %d page", pages)
					}
					break
				}
				if page.Marker == "" || page.Marker == input.Marker {
					t.Fatalf("page %d did not advance marker: %#v", pages, page)
				}
				input.Marker = page.Marker
			}
			for _, key := range want {
				if !seen[key] {
					t.Errorf("paginated %s keys missing %q", section, key)
				}
			}
		})
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
