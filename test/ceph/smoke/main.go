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
	fmt.Printf("created and retrieved bucket %q\n", name)
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
