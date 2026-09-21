package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sj14/rgw-go/rgw"
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
		uid:    uniqueResourceName(t, "rgw-go-admin-integration-user"),
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
