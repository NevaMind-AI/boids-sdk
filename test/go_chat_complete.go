package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	boids "github.com/NevaMind-AI/boids-sdk/go"
)

type requestRecord struct {
	path          string
	authorization string
	body          map[string]any
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("go chat/complete test passed")
}

func run() error {
	records := []requestRecord{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		records = append(records, requestRecord{
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
			body:          body,
		})

		if r.URL.Path != "/chat/complete" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "bad authorization", http.StatusUnauthorized)
			return
		}

		streaming, _ := body["stream"].(bool)
		if streaming {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: chat.delta\ndata: {\"delta\": \"Hello\"}\n\n")
			fmt.Fprint(w, "event: chat.completed\ndata: {\"id\": \"chat_123\", \"output_text\": \"Hello\"}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chat_123",
			"message": map[string]any{"role": "assistant", "content": "Hello"},
		})
	}))
	defer server.Close()

	client := boids.NewClient("test-key", boids.WithBaseURL(server.URL))
	messages := []map[string]string{{"role": "user", "content": "Say hello"}}

	response, err := client.CompleteChat(context.Background(), boids.ChatCompleteRequest{
		Model:    "agent:test",
		Messages: messages,
		Extra:    map[string]any{"temperature": 0.2},
	})
	if err != nil {
		return err
	}
	responseMap, ok := response.(map[string]any)
	if !ok || responseMap["id"] != "chat_123" {
		return fmt.Errorf("unexpected response: %#v", response)
	}
	message, ok := responseMap["message"].(map[string]any)
	if !ok || message["content"] != "Hello" {
		return fmt.Errorf("unexpected message: %#v", responseMap["message"])
	}

	stream, err := client.StreamChatComplete(context.Background(), boids.ChatCompleteRequest{
		Model:    "agent:test",
		Messages: messages,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	eventNames := []string{}
	var text strings.Builder
	for stream.Next() {
		event := stream.Event()
		eventNames = append(eventNames, event.Event)
		data, ok := event.Data.(map[string]any)
		if !ok {
			continue
		}
		if delta, ok := data["delta"].(string); ok {
			text.WriteString(delta)
		}
	}
	if err := stream.Err(); err != nil {
		return err
	}
	if strings.Join(eventNames, ",") != "chat.delta,chat.completed" {
		return fmt.Errorf("unexpected events: %v", eventNames)
	}
	if text.String() != "Hello" {
		return fmt.Errorf("unexpected stream text: %q", text.String())
	}

	if len(records) != 2 {
		return fmt.Errorf("expected 2 requests, got %d", len(records))
	}
	if records[0].path != "/chat/complete" || records[1].path != "/chat/complete" {
		return fmt.Errorf("unexpected paths: %s, %s", records[0].path, records[1].path)
	}
	if records[0].authorization != "Bearer test-key" {
		return fmt.Errorf("unexpected authorization: %s", records[0].authorization)
	}
	if records[0].body["model"] != "agent:test" {
		return fmt.Errorf("unexpected model: %#v", records[0].body["model"])
	}
	if records[0].body["temperature"] != 0.2 {
		return fmt.Errorf("unexpected temperature: %#v", records[0].body["temperature"])
	}
	if records[0].body["stream"] != false || records[1].body["stream"] != true {
		return fmt.Errorf("unexpected stream flags: %#v %#v", records[0].body["stream"], records[1].body["stream"])
	}

	return nil
}
