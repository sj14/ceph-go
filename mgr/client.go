// Package mgr provides a Go client for Ceph Dashboard APIs exposed by the Ceph
// Manager, focused on RGW management and related cluster information.
package mgr

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	mediaTypeV1_0      = "application/vnd.ceph.api.v1.0+json"
	mediaTypeV1_1      = "application/vnd.ceph.api.v1.1+json"
	defaultHTTPTimeout = 30 * time.Second
	maxResponseBody    = 4 << 20
)

// Client calls the Ceph Dashboard API.
//
// A Client is safe for concurrent use. Its exported fields must not be
// modified while requests are in flight.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
	userAgent  string
}

// Option configures a Client.
type Option func(*Client) error

// WithBearerToken configures the JWT returned by the Ceph Dashboard auth API.
func WithBearerToken(token string) Option {
	return func(client *Client) error {
		if strings.TrimSpace(token) == "" {
			return errors.New("mgr: bearer token must not be empty")
		}
		client.token = token
		return nil
	}
}

// WithHTTPClient configures the HTTP client used for requests. The supplied
// client replaces the default client, including its 30-second timeout.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) error {
		if httpClient == nil {
			return errors.New("mgr: HTTP client must not be nil")
		}
		client.httpClient = httpClient
		return nil
	}
}

// WithUserAgent configures the User-Agent header. Empty values disable it.
func WithUserAgent(userAgent string) Option {
	return func(client *Client) error {
		client.userAgent = userAgent
		return nil
	}
}

// NewClient constructs a client for a Ceph Dashboard base URL. The base URL
// may include the dashboard's configured path prefix. The default HTTP client
// has a 30-second total request timeout.
func NewClient(baseURL string, options ...Option) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("mgr: parse base URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("mgr: base URL scheme must be http or https")
	}
	if parsedURL.Host == "" {
		return nil, errors.New("mgr: base URL must include a host")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("mgr: base URL must not include a query or fragment")
	}

	client := &Client{
		baseURL:    parsedURL,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		userAgent:  "ceph-go/mgr",
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("mgr: option must not be nil")
		}
		if err := option(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (client *Client) endpoint(path string) *url.URL {
	endpoint := *client.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint.RawPath = ""
	return &endpoint
}

func (client *Client) do(request *http.Request) ([]byte, error) {
	if request.Header.Get("Accept") == "" {
		request.Header.Set("Accept", mediaTypeV1_0)
	}
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}
	if client.userAgent != "" {
		request.Header.Set("User-Agent", client.userAgent)
	}

	httpResponse, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("mgr: %s %s: %w", request.Method, request.URL.Path, err)
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBody+1))
	if err != nil {
		return nil, fmt.Errorf("mgr: read %s %s response: %w", request.Method, request.URL.Path, err)
	}
	if len(body) > maxResponseBody {
		return nil, fmt.Errorf("mgr: %s %s response exceeds %d bytes", request.Method, request.URL.Path, maxResponseBody)
	}

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return nil, &APIError{
			StatusCode: httpResponse.StatusCode,
			Method:     request.Method,
			Path:       request.URL.Path,
			Body:       string(body),
		}
	}

	return body, nil
}

// APIError is returned for non-2xx Ceph Dashboard responses.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (err *APIError) Error() string {
	message := strings.TrimSpace(err.Body)
	if message == "" {
		message = http.StatusText(err.StatusCode)
	}
	return fmt.Sprintf("mgr: %s %s returned %d: %s", err.Method, err.Path, err.StatusCode, message)
}
