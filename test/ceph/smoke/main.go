package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	rgw "github.com/sj14/rgw-go"
)

const mediaType = "application/vnd.ceph.api.v1.0+json"

func main() {
	baseURL := env("CEPH_DASHBOARD_URL", "http://127.0.0.1:8443")
	username := env("CEPH_DASHBOARD_USER", "admin")
	password := env("CEPH_DASHBOARD_PASSWORD", "admin")

	token, err := authenticate(baseURL, username, password)
	if err != nil {
		fatal(err)
	}
	client, err := rgw.NewClient(baseURL, rgw.WithBearerToken(token))
	if err != nil {
		fatal(err)
	}

	name := fmt.Sprintf("rgw-go-smoke-%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID := fmt.Sprintf("rgw-go-smoke-user-%d", time.Now().UnixNano())
	generateKey := false
	userCreated := false
	defer func() {
		if userCreated {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cleanupCancel()
			_ = client.DeleteUser(cleanupCtx, rgw.DeleteUserRequest{UID: userID})
		}
	}()
	user, err := client.CreateUser(ctx, rgw.CreateUserRequest{
		UID:         userID,
		DisplayName: "rgw-go smoke user",
		GenerateKey: &generateKey,
	})
	if err != nil {
		fatal(err)
	}
	userCreated = true
	if user.UID != userID {
		fatal(fmt.Errorf("created unexpected user: uid=%q", user.UID))
	}
	user, err = client.GetUser(ctx, rgw.GetUserRequest{UID: userID})
	if err != nil {
		fatal(err)
	}
	if user.DisplayName != "rgw-go smoke user" || user.Stats == nil {
		fatal(fmt.Errorf("retrieved unexpected user: uid=%q display_name=%q stats=%v",
			user.UID, user.DisplayName, user.Stats))
	}
	users, err := client.ListUsers(ctx, rgw.ListUsersRequest{})
	if err != nil {
		fatal(err)
	}
	foundUser := false
	for _, listedUser := range users {
		if listedUser.UID == userID {
			foundUser = true
			break
		}
	}
	if !foundUser {
		fatal(fmt.Errorf("created user %q is missing from user list", userID))
	}
	email := "smoke@example.invalid"
	user, err = client.UpdateUser(ctx, rgw.UpdateUserRequest{UID: userID, Email: &email})
	if err != nil {
		fatal(err)
	}
	if user.Email != email {
		fatal(fmt.Errorf("updated unexpected user: uid=%q email=%q", user.UID, user.Email))
	}
	if err := client.DeleteUser(ctx, rgw.DeleteUserRequest{UID: userID}); err != nil {
		fatal(err)
	}
	userCreated = false

	if err := client.CreateBucket(ctx, rgw.CreateBucketRequest{Name: name, UID: "rgw-go-test"}); err != nil {
		fatal(err)
	}
	bucket, err := client.GetBucket(ctx, rgw.GetBucketRequest{Name: name})
	if err != nil {
		fatal(err)
	}
	if bucket.Name != name || bucket.Owner != "rgw-go-test" {
		fatal(fmt.Errorf("unexpected bucket: name=%q owner=%q", bucket.Name, bucket.Owner))
	}
	fmt.Printf("completed user CRUD for %q and created and retrieved bucket %q\n", userID, name)
}

func authenticate(baseURL, username, password string) (string, error) {
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/auth", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", mediaType)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("authentication returned %s", response.Status)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Token == "" {
		return "", fmt.Errorf("authentication response contains no token")
	}
	return result.Token, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
