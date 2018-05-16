// Package luno is a wrapper for the Luno API.
package luno

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Error is a Luno API error.
type Error struct {
	// Code can be used to identify errors even if the error message is
	// localised.
	Code string `json:"error_code"`

	// Message may be localised for authenticated API calls.
	Message string `json:"error"`
}

func (e *Error) Error() string {
	return e.Message
}

// Client is a Luno API client.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	apiKeyID     string
	apiKeySecret string
}

const defaultBaseURL = "https://api.mybitx.com"

const defaultTimeout = 10 * time.Second

// NewClient creates a new Luno API client with the default base URL.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		baseURL:    defaultBaseURL,
	}
}

// SetAuth provides the client with an API key and secret.
func (cl *Client) SetAuth(apiKeyID, apiKeySecret string) error {
	if apiKeyID == "" || apiKeySecret == "" {
		return errors.New("luno: no credentials provided")
	}
	cl.apiKeyID = apiKeyID
	cl.apiKeySecret = apiKeySecret
	return nil
}

// SetBaseURL overrides the default base URL. For internal use.
func (cl *Client) SetBaseURL(baseURL string) {
	cl.baseURL = strings.TrimRight(baseURL, "/")
}

// SetTimeout sets the timeout for requests made by this client.
func (cl *Client) SetTimeout(timeout time.Duration) {
	cl.httpClient.Timeout = timeout
}

func (cl *Client) do(ctx context.Context, method, path string,
	req, res interface{}, auth bool) error {

	url := cl.baseURL + "/" + strings.TrimLeft(path, "/")

	var contentType string
	var body io.Reader
	if req != nil {
		values, err := makeURLValues(req)
		if err != nil {
			return err
		}
		if strings.Contains(path, "{id}") {
			url = strings.Replace(url, "{id}", values.Get("id"), -1)
			values.Del("id")
		}
		if method == http.MethodGet {
			url = url + "?" + values.Encode()
		} else {
			body = strings.NewReader(values.Encode())
			contentType = "application/x-www-form-urlencoded"
		}
	}

	httpReq, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	httpReq = httpReq.WithContext(ctx)
	httpReq.Header.Set("User-Agent", makeUserAgent())
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	if auth {
		httpReq.SetBasicAuth(cl.apiKeyID, cl.apiKeySecret)
	}

	if method != http.MethodGet {
		httpReq.Header.Set("content-type", "application/x-www-form-urlencoded")
	}

	httpRes, err := cl.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpRes.Body.Close()

	// TODO: Handle 429
	if httpRes.StatusCode != http.StatusOK {
		var e Error
		if err := json.NewDecoder(httpRes.Body).Decode(&e); err != nil {
			return fmt.Errorf("luno: error decoding response (%d %s)",
				httpRes.StatusCode, http.StatusText(httpRes.StatusCode))
		}
		return &e
	}

	return json.NewDecoder(httpRes.Body).Decode(res)
}

func makeUserAgent() string {
	return fmt.Sprintf("LunoGoSDK/%s %s %s %s",
		Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}