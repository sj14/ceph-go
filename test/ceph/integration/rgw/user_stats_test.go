package integration

import (
	"errors"
	"net/http"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWRefreshUserStats(t *testing.T) {
	t.Parallel()
	for _, accountMember := range []bool{false, true} {
		name := "user"
		if accountMember {
			name = "account-member"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := rgwIntegrationClient(t)
			ctx := integrationContext(t)
			var uid, accountID string
			var bucket *rgwBucketFixture
			if accountMember {
				accountID, uid, bucket = rgwAccountBucketFixture(t, client, ctx)
			} else {
				user, _ := createRGWUserFixture(t, client, ctx)
				uid = user.uid
				bucket = createRGWBucketFixture(t, client, ctx)
			}
			payload := []byte("statistics integration test")
			resource := bucket.name + "/first-object"
			requireS3Success(t, ctx, http.MethodPut, resource, payload)
			wantCount := int64(1)
			if accountMember {
				for _, selectedBucket := range []*rgwBucketFixture{bucket, createRGWBucketFixture(t, client, ctx)} {
					if selectedBucket != bucket {
						requireS3Success(t, ctx, http.MethodPut, selectedBucket.name+"/second-object", payload)
						wantCount++
					}
					if err := client.LinkBucketToAccount(ctx, rgw.LinkBucketToAccountRequest{
						Name: selectedBucket.name, AccountID: accountID,
					}); err != nil {
						t.Fatal(err)
					}
				}
			} else if err := client.LinkBucketToUser(ctx, rgw.LinkBucketToUserRequest{Name: bucket.name, UID: uid}); err != nil {
				t.Fatal(err)
			}
			for _, invalidUID := range []string{"", " "} {
				_, err := client.GetUser(ctx, rgw.GetUserRequest{UID: invalidUID, RefreshStats: new(true)})
				var apiErr *rgw.APIError
				if err == nil || errors.As(err, &apiErr) {
					t.Fatalf("missing UID error = %v, want local validation error", err)
				}
				_, err = client.GetUserByAccessKey(ctx, rgw.GetUserByAccessKeyRequest{AccessKey: invalidUID, RefreshStats: new(true)})
				if err == nil || errors.As(err, &apiErr) {
					t.Fatalf("missing access key error = %v, want local validation error", err)
				}
			}
			keys, err := client.CreateS3Key(ctx, rgw.CreateS3KeyRequest{UID: uid})
			if err != nil {
				t.Fatal(err)
			}
			if len(keys) != 1 {
				t.Fatalf("created keys = %#v, want one key", keys)
			}
			for _, deleted := range []bool{false, true} {
				if deleted {
					requireS3Success(t, ctx, http.MethodDelete, resource, nil)
					wantCount--
				}
				var refreshed rgw.User
				if deleted {
					refreshed, err = client.GetUserByAccessKey(ctx, rgw.GetUserByAccessKeyRequest{
						AccessKey: keys[0].AccessKey, Stats: new(true), RefreshStats: new(true),
					})
				} else {
					refreshed, err = client.GetUser(ctx, rgw.GetUserRequest{
						UID: uid, Stats: new(true), RefreshStats: new(true),
					})
				}
				if err != nil {
					t.Fatal(err)
				}
				stats := refreshed.Stats
				if refreshed.ID != uid || stats == nil || stats.NumObjects != wantCount || stats.Size != wantCount*int64(len(payload)) {
					t.Fatalf("refreshed user = %#v, want user %s with %d objects", refreshed, uid, wantCount)
				}
				byUID, err := client.GetUser(ctx, rgw.GetUserRequest{UID: uid, Stats: new(true)})
				if err != nil {
					t.Fatal(err)
				}
				byKey, err := client.GetUserByAccessKey(ctx, rgw.GetUserByAccessKeyRequest{AccessKey: keys[0].AccessKey, Stats: new(true)})
				if err != nil {
					t.Fatal(err)
				}
				if byUID.Stats == nil || byKey.Stats == nil || *byUID.Stats != *stats || *byKey.Stats != *stats {
					t.Fatalf("stored statistics differ from synchronization: UID=%#v key=%#v sync=%#v", byUID.Stats, byKey.Stats, stats)
				}
			}
			for _, byAccessKey := range []bool{false, true} {
				if byAccessKey {
					requireS3Success(t, ctx, http.MethodDelete, resource, nil)
					wantCount--
				} else {
					requireS3Success(t, ctx, http.MethodPut, resource, payload)
					wantCount++
				}
				var refreshed rgw.User
				if byAccessKey {
					refreshed, err = client.GetUserByAccessKey(ctx, rgw.GetUserByAccessKeyRequest{
						AccessKey: keys[0].AccessKey, Stats: new(false), RefreshStats: new(true),
					})
				} else {
					refreshed, err = client.GetUser(ctx, rgw.GetUserRequest{UID: uid, RefreshStats: new(true)})
				}
				if err != nil {
					t.Fatal(err)
				}
				if refreshed.ID != uid || refreshed.Stats != nil {
					t.Fatalf("refresh without statistics returned %#v", refreshed)
				}
				stored, err := client.GetUser(ctx, rgw.GetUserRequest{UID: uid, Stats: new(true), RefreshStats: new(false)})
				if err != nil {
					t.Fatal(err)
				}
				if stored.Stats == nil || stored.Stats.NumObjects != wantCount || stored.Stats.Size != wantCount*int64(len(payload)) {
					t.Fatalf("stored statistics = %#v, want %d objects", stored.Stats, wantCount)
				}
			}
		})
	}
}
