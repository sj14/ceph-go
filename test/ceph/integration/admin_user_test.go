package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sj14/rgw-go/admin"
)

type adminUserFixture struct {
	client  *admin.Client
	uid     string
	deleted bool
}

func adminIntegrationClient(t *testing.T) *admin.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Ceph integration test in short mode")
	}
	client, err := admin.NewClient(
		environment("CEPH_RGW_URL", "http://127.0.0.1:8000"),
		environment("CEPH_ADMIN_RGW_ACCESS_KEY", "RGWGOADMINACCESSKEY"),
		environment("CEPH_ADMIN_RGW_SECRET_KEY", "rgw-go-admin-secret-key-for-integration-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func createAdminUserFixture(t *testing.T, client *admin.Client, ctx context.Context) (*adminUserFixture, admin.User) {
	t.Helper()
	fixture := &adminUserFixture{
		client: client,
		uid:    uniqueResourceName(t, "rgw-go-admin-integration-user"),
	}
	user, err := client.CreateUser(ctx, admin.CreateUserRequest{
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
		if err := fixture.client.DeleteUser(cleanupCtx, admin.DeleteUserRequest{UID: fixture.uid}); err != nil {
			t.Errorf("cleanup Admin Ops user %q: %v", fixture.uid, err)
		}
	})
	return fixture, user
}

func TestAdminCreateUser(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	fixture, user := createAdminUserFixture(t, client, integrationContext(t))
	if user.ID != fixture.uid || user.FullUserID != fixture.uid ||
		user.DisplayName != "Admin API Integration User" || user.MaxBuckets != 37 ||
		user.Type != admin.UserTypeRGW || user.CreateDate.IsZero() || len(user.Keys) != 0 {
		t.Fatalf("created Admin Ops user = %#v", user)
	}
}

func TestAdminGetUser(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	user, err := client.GetUser(ctx, admin.GetUserRequest{UID: fixture.uid, Stats: new(true)})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != fixture.uid || user.Stats == nil || user.Stats.NumObjects != 0 {
		t.Fatalf("Admin Ops user = %#v", user)
	}
}

func TestAdminUpdateUser(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	user, err := client.UpdateUser(ctx, admin.UpdateUserRequest{
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

func TestAdminDeleteUser(t *testing.T) {
	t.Parallel()

	client := adminIntegrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createAdminUserFixture(t, client, ctx)
	if err := client.DeleteUser(ctx, admin.DeleteUserRequest{UID: fixture.uid}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetUser(ctx, admin.GetUserRequest{UID: fixture.uid})
	if !errors.Is(err, admin.ErrNoSuchUser) {
		t.Fatalf("GetUser after Admin Ops delete error = %v, want NoSuchUser", err)
	}
}
