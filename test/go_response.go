package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	fmt.Println("go responses test passed")
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

		if r.URL.Path != "/responses" {
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
			fmt.Fprint(w, "event: response.output_text.delta\ndata: {\"delta\": \"Hello\"}\n\n")
			fmt.Fprint(w, "event: response.completed\ndata: {\"id\": \"resp_123\", \"output_text\": \"Hello\"}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "resp_123",
			"output_text": "Hello",
		})
	}))
	defer server.Close()

	client := boids.NewClient("test-key", boids.WithBaseURL(server.URL))

	response, err := client.CreateResponse(context.Background(), boids.ResponseRequest{
		Model: "agent:test",
		Input: "Say hello",
		Extra: map[string]any{"temperature": 0.2},
	})
	if err != nil {
		return err
	}
	if encoded, err := json.Marshal(response); err == nil {
		fmt.Printf("[sdk create] %s\n", encoded)
	}
	responseMap, ok := response.(map[string]any)
	if !ok || responseMap["id"] != "resp_123" {
		return fmt.Errorf("unexpected response: %#v", response)
	}
	if responseMap["output_text"] != "Hello" {
		return fmt.Errorf("unexpected output_text: %#v", responseMap["output_text"])
	}

	stream, err := client.StreamResponse(context.Background(), boids.ResponseRequest{
		Model: "agent:test",
		Input: "Say hello",
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	eventNames := []string{}
	events := []boids.ResponseEvent{}
	var text strings.Builder
	for stream.Next() {
		event := stream.Event()
		eventNames = append(eventNames, event.Event)
		events = append(events, event)
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
	if encoded, err := json.Marshal(events); err == nil {
		fmt.Printf("[sdk stream] %s\n", encoded)
	}
	if strings.Join(eventNames, ",") != "response.output_text.delta,response.completed" {
		return fmt.Errorf("unexpected events: %v", eventNames)
	}
	if text.String() != "Hello" {
		return fmt.Errorf("unexpected stream text: %q", text.String())
	}

	if len(records) != 2 {
		return fmt.Errorf("expected 2 requests, got %d", len(records))
	}
	if records[0].path != "/responses" || records[1].path != "/responses" {
		return fmt.Errorf("unexpected paths: %s, %s", records[0].path, records[1].path)
	}
	if records[0].authorization != "Bearer test-key" {
		return fmt.Errorf("unexpected authorization: %s", records[0].authorization)
	}
	if records[0].body["model"] != "agent:test" {
		return fmt.Errorf("unexpected model: %#v", records[0].body["model"])
	}
	if records[0].body["input"] != "Say hello" {
		return fmt.Errorf("unexpected input: %#v", records[0].body["input"])
	}
	if records[0].body["temperature"] != 0.2 {
		return fmt.Errorf("unexpected temperature: %#v", records[0].body["temperature"])
	}
	if records[0].body["stream"] != false || records[1].body["stream"] != true {
		return fmt.Errorf("unexpected stream flags: %#v %#v", records[0].body["stream"], records[1].body["stream"])
	}

	cliRecords := []requestRecord{}
	cliServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		cliRecords = append(cliRecords, requestRecord{
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
			body:          body,
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "resp_cli_123",
			"output_text": "Hello from CLI",
		})
	}))
	defer cliServer.Close()

	cliOut, err := runCLI(
		"responses",
		"create",
		"-api-key", "test-key",
		"-base-url", cliServer.URL,
		"-model", "agent:test",
		"-input", "Say hello",
		"-json",
	)
	if err != nil {
		return err
	}
	var cliResponse map[string]any
	if err := json.Unmarshal(cliOut, &cliResponse); err != nil {
		return fmt.Errorf("cli output not json: %v (%s)", err, cliOut)
	}
	compact, _ := json.Marshal(cliResponse)
	fmt.Printf("[cli responses create] %s\n", compact)
	if cliResponse["id"] != "resp_cli_123" {
		return fmt.Errorf("unexpected cli response: %#v", cliResponse)
	}
	if len(cliRecords) != 1 {
		return fmt.Errorf("expected 1 cli request, got %d", len(cliRecords))
	}
	if cliRecords[0].path != "/responses" {
		return fmt.Errorf("unexpected cli path: %s", cliRecords[0].path)
	}
	if cliRecords[0].authorization != "Bearer test-key" {
		return fmt.Errorf("unexpected cli authorization: %s", cliRecords[0].authorization)
	}
	if cliRecords[0].body["model"] != "agent:test" {
		return fmt.Errorf("unexpected cli model: %#v", cliRecords[0].body["model"])
	}
	if cliRecords[0].body["input"] != "Say hello" {
		return fmt.Errorf("unexpected cli input: %#v", cliRecords[0].body["input"])
	}
	if cliRecords[0].body["stream"] != false {
		return fmt.Errorf("unexpected cli stream flag: %#v", cliRecords[0].body["stream"])
	}

	return nil
}

func runCLI(args ...string) ([]byte, error) {
	cmd := exec.Command("go", append([]string{"run", "./cmd/boids"}, args...)...)
	cmd.Dir = "../go"
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cli failed: %v: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}
