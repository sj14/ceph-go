package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	rgw "github.com/sj14/rgw-go"
)

const mediaType = "application/vnd.ceph.api.v1.0+json"

func integrationClient(t *testing.T) *rgw.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Ceph integration test in short mode")
	}

	baseURL := environment("CEPH_DASHBOARD_URL", "http://127.0.0.1:8443")
	token, err := authenticate(
		baseURL,
		environment("CEPH_DASHBOARD_USER", "admin"),
		environment("CEPH_DASHBOARD_PASSWORD", "admin"),
	)
	if err != nil {
		t.Fatal(err)
	}
	client, err := rgw.NewClient(baseURL, rgw.WithBearerToken(token))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func integrationContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func uniqueResourceName(t *testing.T, prefix string) string {
	t.Helper()
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s-%x", prefix, suffix)
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

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
