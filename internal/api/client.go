package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	httpClient      *http.Client
	baseURLOverride string
}

type Request struct {
	App         string
	Env         string
	Method      string
	Path        string
	Payload     any
	Body        io.Reader
	ContentType string
	Accept      string
	Token       string
}

func NewClient(httpClient *http.Client, baseURLOverride string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, baseURLOverride: baseURLOverride}
}

func (c *Client) endpoint(app, env, path string) (*url.URL, error) {
	baseURL := c.baseURLOverride
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.slingrs.io/%s/runtime/api", app, env)
	}
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %w", err)
	}
	return endpoint, nil
}

func (c *Client) Do(ctx context.Context, request Request) ([]byte, error) {
	endpoint, err := c.endpoint(request.App, request.Env, request.Path)
	if err != nil {
		return nil, err
	}
	var body io.Reader
	if request.Body != nil {
		body = request.Body
	} else if request.Payload != nil {
		encoded, err := json.Marshal(request.Payload)
		if err != nil {
			return nil, fmt.Errorf("encode request payload: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, request.Method, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	accept := request.Accept
	if accept == "" {
		accept = "application/json"
	}
	req.Header.Set("Accept", accept)
	contentType := request.ContentType
	if contentType == "" && request.Payload != nil {
		contentType = "application/json"
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if request.Token != "" {
		req.Header.Set("token", request.Token)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read API response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var apiError struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &apiError)
		if apiError.Message != "" {
			return nil, fmt.Errorf("API error (%s): %s", response.Status, apiError.Message)
		}
		return nil, fmt.Errorf("API error: %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}
