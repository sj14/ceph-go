package integration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	rgw "github.com/sj14/rgw-go"
)

type userFixture struct {
	client  *rgw.Client
	uid     string
	email   string
	deleted bool
}

func createUserFixture(t *testing.T, client *rgw.Client, ctx context.Context) (*userFixture, rgw.User) {
	t.Helper()

	fixture := &userFixture{
		client: client,
		uid:    uniqueResourceName(t, "rgw-go-integration-user"),
	}
	fixture.email = fixture.uid + "@example.invalid"
	user, err := client.CreateUser(ctx, rgw.CreateUserRequest{
		UID:         fixture.uid,
		DisplayName: "Integration User",
		Email:       &fixture.email,
		MaxBuckets:  new(int64(1234)),
		System:      new(false),
		Suspended:   new(false),
		GenerateKey: new(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if fixture.deleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := fixture.client.DeleteUser(cleanupCtx, rgw.DeleteUserRequest{UID: fixture.uid}); err != nil {
			t.Errorf("cleanup user %q: %v", fixture.uid, err)
		}
	})
	return fixture, user
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	fixture, user := createUserFixture(t, client, integrationContext(t))

	if user.UID != fixture.uid || user.DisplayName != "Integration User" ||
		user.Email != fixture.email || user.MaxBuckets != 1234 {
		t.Fatalf("created user = %#v", user)
	}
	if len(user.Keys) != 0 {
		t.Fatalf("created user keys = %#v, want none", user.Keys)
	}
}

func TestGetUser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	user, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	if err != nil {
		t.Fatal(err)
	}
	if user.UID != fixture.uid || user.Stats == nil || user.Stats.NumObjects != 0 {
		t.Fatalf("user = %#v", user)
	}

	user, err = client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid, Stats: new(false)})
	if err != nil {
		t.Fatal(err)
	}
	if user.Stats != nil {
		t.Fatalf("user stats = %#v, want omitted stats", user.Stats)
	}
}

func TestListUsers(t *testing.T) {
	// Keep this test serial: with detailed=true, Ceph first lists all UIDs and
	// then fetches each user's details separately. If a parallel test deletes
	// any listed user between those steps, the entire request fails with
	// NoSuchUser, even though this test's own user still exists.
	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	users, err := client.ListUsers(ctx, rgw.ListUsersRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range users {
		if user.UID == fixture.uid {
			if user.Stats != nil {
				t.Fatalf("listed user stats = %#v, want omitted stats", user.Stats)
			}
			return
		}
	}
	t.Fatalf("created user %q is missing from ListUsers", fixture.uid)
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	user, err := client.UpdateUser(ctx, rgw.UpdateUserRequest{
		UID:         fixture.uid,
		DisplayName: new("Updated Integration User"),
		Email:       new(""),
		MaxBuckets:  new(int64(0)),
		System:      new(false),
		Suspended:   new(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.DisplayName != "Updated Integration User" || user.Email != "" || user.MaxBuckets != 0 ||
		user.System || user.Suspended != 1 {
		t.Fatalf("updated user = %#v", user)
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()

	client := integrationClient(t)
	ctx := integrationContext(t)
	fixture, _ := createUserFixture(t, client, ctx)

	if err := client.DeleteUser(ctx, rgw.DeleteUserRequest{UID: fixture.uid}); err != nil {
		t.Fatal(err)
	}
	fixture.deleted = true

	_, err := client.GetUser(ctx, rgw.GetUserRequest{UID: fixture.uid})
	var apiError *rgw.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusInternalServerError ||
		!strings.Contains(apiError.Body, "NoSuchUser") {
		t.Fatalf("GetUser after delete error = %v, want Dashboard HTTP 500 containing NoSuchUser", err)
	}
}
