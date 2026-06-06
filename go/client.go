package boids

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	DefaultBaseURL = "https://api.boids.so/v1"
	userAgent     = "boids-go/0.1.1"
)

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Headers    http.Header
}

type Option func(*Client)

func WithBaseURL(baseURL string) Option {
	return func(client *Client) {
		client.BaseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		client.HTTPClient = httpClient
	}
}

func WithHeader(key, value string) Option {
	return func(client *Client) {
		client.Headers.Set(key, value)
	}
}

func NewClient(apiKey string, options ...Option) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("BOIDS_API_KEY")
	}

	client := &Client{
		APIKey:     apiKey,
		BaseURL:    DefaultBaseURL,
		HTTPClient: http.DefaultClient,
		Headers:    make(http.Header),
	}

	for _, option := range options {
		option(client)
	}

	client.BaseURL = strings.TrimRight(client.BaseURL, "/")
	return client
}

type ResponseRequest struct {
	Model              string
	Input              any
	Stream             bool
	PreviousResponseID string
	Extra              map[string]any
}

type ChatCompleteRequest struct {
	Model    string
	Messages any
	Stream   bool
	Extra    map[string]any
}

type MarketSearchRequest struct {
	Query string
	Limit int
	Extra map[string]any
}

type ResponseEvent struct {
	Event string `json:"event,omitempty"`
	Data  any    `json:"data"`
	Raw   string `json:"raw"`
}

type APIError struct {
	StatusCode int
	Body       string
}

func (err *APIError) Error() string {
	return fmt.Sprintf("Boids API error %d: %s", err.StatusCode, err.Body)
}

func (client *Client) CreateResponse(ctx context.Context, request ResponseRequest) (any, error) {
	response, err := client.postResponse(ctx, request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var output any
	if err := json.NewDecoder(response.Body).Decode(&output); err != nil {
		return nil, err
	}

	return output, nil
}

func (client *Client) CompleteChat(ctx context.Context, request ChatCompleteRequest) (any, error) {
	response, err := client.postChatComplete(ctx, request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var output any
	if err := json.NewDecoder(response.Body).Decode(&output); err != nil {
		return nil, err
	}

	return output, nil
}

func (client *Client) StreamResponse(ctx context.Context, request ResponseRequest) (*ResponseStream, error) {
	request.Stream = true
	response, err := client.postResponse(ctx, request)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	return &ResponseStream{
		scanner: scanner,
		closer:  response.Body,
	}, nil
}

func (client *Client) StreamChatComplete(ctx context.Context, request ChatCompleteRequest) (*ResponseStream, error) {
	request.Stream = true
	response, err := client.postChatComplete(ctx, request)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	return &ResponseStream{
		scanner: scanner,
		closer:  response.Body,
	}, nil
}

func (client *Client) SearchMarket(ctx context.Context, request MarketSearchRequest) (any, error) {
	limit := request.Limit
	if limit == 0 {
		limit = 5
	}

	body := make(map[string]any, len(request.Extra)+2)
	for key, value := range request.Extra {
		if value != nil {
			body[key] = value
		}
	}
	body["query"] = request.Query
	body["limit"] = limit

	response, err := client.post(ctx, "/market/search", body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var output any
	if err := json.NewDecoder(response.Body).Decode(&output); err != nil {
		return nil, err
	}

	return output, nil
}

func (client *Client) postResponse(ctx context.Context, responseRequest ResponseRequest) (*http.Response, error) {
	if client.APIKey == "" {
		return nil, fmt.Errorf("missing BOIDS_API_KEY or API key")
	}

	body := make(map[string]any, len(responseRequest.Extra)+3)
	for key, value := range responseRequest.Extra {
		if value != nil {
			body[key] = value
		}
	}
	body["model"] = responseRequest.Model
	body["input"] = responseRequest.Input
	body["stream"] = responseRequest.Stream
	if responseRequest.PreviousResponseID != "" {
		body["previous_response_id"] = responseRequest.PreviousResponseID
	}

	return client.post(ctx, "/responses", body)
}

func (client *Client) postChatComplete(ctx context.Context, chatRequest ChatCompleteRequest) (*http.Response, error) {
	if client.APIKey == "" {
		return nil, fmt.Errorf("missing BOIDS_API_KEY or API key")
	}

	body := make(map[string]any, len(chatRequest.Extra)+3)
	for key, value := range chatRequest.Extra {
		if value != nil {
			body[key] = value
		}
	}
	if chatRequest.Model != "" {
		body["model"] = chatRequest.Model
	}
	if chatRequest.Messages != nil {
		body["messages"] = chatRequest.Messages
	}
	body["stream"] = chatRequest.Stream

	return client.post(ctx, "/chat/complete", body)
}

func (client *Client) post(ctx context.Context, path string, body map[string]any) (*http.Response, error) {
	if client.APIKey == "" {
		return nil, fmt.Errorf("missing BOIDS_API_KEY or API key")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.BaseURL+path,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}

	httpRequest.Header.Set("Authorization", "Bearer "+client.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "text/event-stream, application/json")
	httpRequest.Header.Set("User-Agent", userAgent)
	for key, values := range client.Headers {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}

	response, err := client.HTTPClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return nil, &APIError{StatusCode: response.StatusCode, Body: string(body)}
	}

	return response, nil
}

type ResponseStream struct {
	scanner *bufio.Scanner
	closer  io.Closer
	event   ResponseEvent
	err     error
	closed  bool
}

func (stream *ResponseStream) Next() bool {
	if stream.err != nil || stream.closed {
		return false
	}

	var eventName string
	var dataLines []string
	var rawLines []string

	for stream.scanner.Scan() {
		line := strings.TrimSuffix(stream.scanner.Text(), "\r")

		if line == "" {
			event, ok := makeEvent(eventName, dataLines, rawLines)
			if !ok {
				eventName = ""
				dataLines = nil
				rawLines = nil
				continue
			}
			if event.Raw == "[DONE]" {
				_ = stream.Close()
				return false
			}
			stream.event = event
			return true
		}

		rawLines = append(rawLines, line)
		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")

		switch field {
		case "event":
			eventName = value
		case "data":
			dataLines = append(dataLines, value)
		}
	}

	if err := stream.scanner.Err(); err != nil {
		stream.err = err
		_ = stream.Close()
		return false
	}

	event, ok := makeEvent(eventName, dataLines, rawLines)
	if ok && event.Raw != "[DONE]" {
		stream.event = event
		return true
	}

	_ = stream.Close()
	return false
}

func (stream *ResponseStream) Event() ResponseEvent {
	return stream.event
}

func (stream *ResponseStream) Err() error {
	return stream.err
}

func (stream *ResponseStream) Close() error {
	if stream.closed {
		return nil
	}
	stream.closed = true
	return stream.closer.Close()
}

func makeEvent(eventName string, dataLines []string, rawLines []string) (ResponseEvent, bool) {
	if len(dataLines) == 0 {
		return ResponseEvent{}, false
	}

	rawData := strings.Join(dataLines, "\n")
	event := ResponseEvent{
		Event: eventName,
		Raw:   strings.Join(rawLines, "\n"),
	}

	if rawData == "[DONE]" {
		event.Data = rawData
		event.Raw = rawData
		return event, true
	}

	var data any
	if err := json.Unmarshal([]byte(rawData), &data); err != nil {
		data = rawData
	}
	event.Data = data
	return event, true
}
