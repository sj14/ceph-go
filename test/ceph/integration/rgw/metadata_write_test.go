package integration

import (
	"net/http"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWPutMetadata(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	// Ceph v20.2.4's user metadata handler ignores the selected update policy;
	// all three source-verified wire values are accepted and applied.
	for _, policy := range []rgw.MetadataUpdateType{
		rgw.MetadataUpdateAlways, rgw.MetadataUpdateByVersion, rgw.MetadataUpdateByTimestamp,
	} {
		metadata, err := client.GetMetadata(ctx, rgw.GetMetadataRequest{Section: rgw.MetadataSectionUser, Key: fixture.uid})
		if err != nil {
			t.Fatal(err)
		}
		previousVersion := metadata.Version
		metadata.Version.Version++
		name := "Metadata update " + string(policy)
		metadata.Data["display_name"] = name
		update, err := client.PutMetadata(ctx, rgw.PutMetadataRequest{
			Section: rgw.MetadataSectionUser, Key: fixture.uid, Metadata: metadata, UpdateType: policy,
		})
		if err != nil {
			t.Fatal(err)
		}
		if update.Status != rgw.MetadataUpdateApplied || update.Version != previousVersion {
			t.Fatalf("metadata update = %#v, want applied and previous version %#v", update, previousVersion)
		}
		stored, err := client.GetMetadata(ctx, rgw.GetMetadataRequest{Section: rgw.MetadataSectionUser, Key: fixture.uid})
		if err != nil {
			t.Fatal(err)
		}
		if stored.Data["display_name"] != name || stored.Version != metadata.Version {
			t.Fatalf("stored display name/version = %v/%#v, want %s/%#v", stored.Data["display_name"], stored.Version, name, metadata.Version)
		}
		user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
		if err != nil || user.DisplayName != name {
			t.Fatalf("user display name = %q, error = %v", user.DisplayName, err)
		}
	}
}

func TestRGWDeleteMetadata(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if err := client.DeleteMetadata(ctx, rgw.DeleteMetadataRequest{Section: rgw.MetadataSectionUser, Key: fixture.uid}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true
	_, err := client.GetMetadata(ctx, rgw.GetMetadataRequest{Section: rgw.MetadataSectionUser, Key: fixture.uid})
	requireRGWAPIStatus(t, err, http.StatusNotFound)
	if err := client.DeleteMetadata(ctx, rgw.DeleteMetadataRequest{Section: rgw.MetadataSectionUser, Key: fixture.uid}); err == nil {
		t.Fatal("deleting missing metadata succeeded")
	} else {
		requireRGWAPIStatus(t, err, http.StatusNotFound)
	}
}
