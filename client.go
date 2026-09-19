// Package rgw provides a Go client for the RGW management endpoints exposed
// by the Ceph Dashboard API.
package rgw

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultMediaType = "application/vnd.ceph.api.v1.0+json"
	maxResponseBody  = 4 << 20
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
			return errors.New("rgw: bearer token must not be empty")
		}
		client.token = token
		return nil
	}
}

// WithHTTPClient configures the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) error {
		if httpClient == nil {
			return errors.New("rgw: HTTP client must not be nil")
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
// may include the dashboard's configured path prefix.
func NewClient(baseURL string, options ...Option) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("rgw: parse base URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("rgw: base URL scheme must be http or https")
	}
	if parsedURL.Host == "" {
		return nil, errors.New("rgw: base URL must include a host")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("rgw: base URL must not include a query or fragment")
	}

	client := &Client{
		baseURL:    parsedURL,
		httpClient: http.DefaultClient,
		userAgent:  "rgw-go",
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("rgw: option must not be nil")
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

func (client *Client) do(request *http.Request) (*Response, error) {
	request.Header.Set("Accept", defaultMediaType)
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}
	if client.userAgent != "" {
		request.Header.Set("User-Agent", client.userAgent)
	}

	httpResponse, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("rgw: %s %s: %w", request.Method, request.URL.Path, err)
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBody+1))
	if err != nil {
		return nil, fmt.Errorf("rgw: read %s %s response: %w", request.Method, request.URL.Path, err)
	}
	if len(body) > maxResponseBody {
		return nil, fmt.Errorf("rgw: %s %s response exceeds %d bytes", request.Method, request.URL.Path, maxResponseBody)
	}

	response := &Response{
		StatusCode: httpResponse.StatusCode,
		Header:     httpResponse.Header.Clone(),
		Body:       body,
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return response, &APIError{
			StatusCode: httpResponse.StatusCode,
			Method:     request.Method,
			Path:       request.URL.Path,
			Body:       string(body),
		}
	}

	return response, nil
}

// Response contains the HTTP metadata and raw body returned by Ceph.
// Create-bucket commonly returns JSON null because RGW's S3 PUT response has
// no useful representation.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
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
	return fmt.Sprintf("rgw: %s %s returned %d: %s", err.Method, err.Path, err.StatusCode, message)
}
