package integration

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/sj14/ceph-go/rgw"
)

type rgwUserFixture struct {
	client  *rgw.Client
	uid     string
	deleted bool
}

func createRGWUserFixture(t *testing.T, client *rgw.Client, ctx context.Context) (*rgwUserFixture, rgw.User) {
	t.Helper()
	fixture := &rgwUserFixture{
		client: client,
		uid:    uniqueResourceName(t, "ceph-go-admin-integration-user"),
	}
	user, err := client.CreateUser(ctx, rgw.CreateUserRequest{
		UID:         fixture.uid,
		DisplayName: "Admin API Integration User",
		GenerateKey: new(false),
		MaxBuckets:  new(int64(37)),
		Suspended:   new(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if fixture.deleted {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := fixture.client.DeleteUser(cleanupCtx, rgw.DeleteUserRequest{UID: fixture.uid}); err != nil {
			t.Errorf("cleanup Admin Ops user %q: %v", fixture.uid, err)
		}
	})
	return fixture, user
}

func TestRGWCreateUser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	fixture, user := createRGWUserFixture(t, client, integrationContext(t))
	if user.ID != fixture.uid || user.FullUserID != fixture.uid ||
		user.DisplayName != "Admin API Integration User" || user.MaxBuckets != 37 ||
		user.Type != rgw.UserTypeRGW || user.CreateDate.IsZero() || len(user.Keys) != 0 {
		t.Fatalf("created Admin Ops user = %#v", user)
	}
}

func TestRGWGetUser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid, Stats: new(true)})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != fixture.uid || user.Stats == nil || user.Stats.NumObjects != 0 {
		t.Fatalf("Admin Ops user = %#v", user)
	}
	for _, uid := range []string{"", " "} {
		_, err := client.GetUser(ctx, rgw.GetUserRequest{UID: uid})
		var apiErr *rgw.APIError
		if err == nil || errors.As(err, &apiErr) {
			t.Fatalf("missing UID error = %v, want local validation error", err)
		}
	}
}

func TestRGWGetUserByAccessKey(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	for _, accessKey := range []string{"", " "} {
		_, err := client.GetUserByAccessKey(ctx, rgw.GetUserByAccessKeyRequest{AccessKey: accessKey})
		var apiErr *rgw.APIError
		if err == nil || errors.As(err, &apiErr) {
			t.Fatalf("missing access key error = %v, want local validation error", err)
		}
	}
	// Two independent owners ensure each key selects its own user.
	for range 2 {
		fixture, _ := createRGWUserFixture(t, client, ctx)
		keys, err := client.CreateS3Key(ctx, rgw.CreateS3KeyRequest{UID: fixture.uid})
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 1 {
			t.Fatalf("created keys = %#v, want one key", keys)
		}
		accessKey := keys[0].AccessKey
		for _, stats := range []*bool{nil, new(false), new(true)} {
			user, err := client.GetUserByAccessKey(ctx, rgw.GetUserByAccessKeyRequest{
				AccessKey: accessKey, Stats: stats,
			})
			if err != nil {
				t.Fatal(err)
			}
			if user.ID != fixture.uid || len(user.Keys) != 1 || user.Keys[0] != keys[0] {
				t.Fatalf("user selected by access key = %#v", user)
			}
			wantStats := stats != nil && *stats
			if (user.Stats != nil) != wantStats || (wantStats && user.Stats.NumObjects != 0) {
				t.Fatalf("user statistics = %#v, requested = %v", user.Stats, wantStats)
			}
			byUID, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid, Stats: stats})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(user, byUID) {
				t.Fatalf("key lookup = %#v, UID lookup = %#v", user, byUID)
			}
		}
		if err := client.DeleteS3Key(ctx, rgw.DeleteS3KeyRequest{UID: fixture.uid, AccessKey: accessKey}); err != nil {
			t.Fatal(err)
		}
		_, err = client.GetUserByAccessKey(ctx, rgw.GetUserByAccessKeyRequest{AccessKey: accessKey})
		// Without a matching key, RGWUser::init retains an anonymous UID;
		// RGWAccessKeyPool::init rejects it with EINVAL before info can
		// return NoSuchUser. Preserve Ceph's InvalidArgument response.
		if !errors.Is(err, rgw.ErrInvalidArgument) {
			t.Fatalf("lookup by deleted key error = %v, want InvalidArgument", err)
		}
		requireRGWAPIStatus(t, err, http.StatusBadRequest)
		if _, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid}); err != nil {
			t.Fatalf("UID lookup after key deletion: %v", err)
		}
	}
}

func TestRGWUpdateUser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	user, err := client.UpdateUser(ctx, rgw.UpdateUserRequest{
		UID:         fixture.uid,
		DisplayName: new("Updated Admin API User"),
		Email:       new("admin-api@example.invalid"),
		MaxBuckets:  new(int64(0)),
		Suspended:   new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != fixture.uid || user.DisplayName != "Updated Admin API User" ||
		user.Email != "admin-api@example.invalid" || user.MaxBuckets != 0 || user.Suspended != 1 {
		t.Fatalf("updated Admin Ops user = %#v", user)
	}
}

func TestRGWDeleteUser(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createRGWUserFixture(t, client, ctx)
	if err := client.DeleteUser(ctx, rgw.DeleteUserRequest{UID: fixture.uid}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if !errors.Is(err, rgw.ErrNoSuchUser) {
		t.Fatalf("GetUser after Admin Ops delete error = %v, want NoSuchUser", err)
	}
}

func TestRGWUpdateUserEmailConflict(t *testing.T) {
	t.Parallel()
	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	owner, _ := createRGWUserFixture(t, client, ctx)
	other, _ := createRGWUserFixture(t, client, ctx)
	email := uniqueResourceName(t, "ceph-go-email") + "@example.invalid"
	if _, err := client.UpdateUser(ctx, rgw.UpdateUserRequest{UID: owner.uid, Email: &email}); err != nil {
		t.Fatal(err)
	}
	_, err := client.UpdateUser(ctx, rgw.UpdateUserRequest{UID: other.uid, Email: &email})
	if !errors.Is(err, rgw.ErrEmailExists) {
		t.Fatalf("duplicate email error = %v, want EmailExists", err)
	}
	requireRGWAPIStatus(t, err, http.StatusConflict)
	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: other.uid})
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "" {
		t.Fatalf("rejected duplicate email changed user email: %q", user.Email)
	}
}
