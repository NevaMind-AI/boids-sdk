package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	boids "github.com/boids/boids-go"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		usage()
		return nil
	}

	switch args[0] {
	case "ask":
		return ask(args[1:])
	case "responses":
		if len(args) > 1 && args[1] == "create" {
			return createResponse(args[2:])
		}
	}

	return shortcut(args)
}

func shortcut(args []string) error {
	flags := flag.NewFlagSet("boids", flag.ContinueOnError)
	apiKey := flags.String("api-key", os.Getenv("BOIDS_API_KEY"), "Boids API key")
	baseURL := flags.String("base-url", boids.DefaultBaseURL, "Boids API base URL")
	stream := flags.Bool("stream", true, "Stream response events")
	noStream := flags.Bool("no-stream", false, "Disable response streaming")
	jsonOutput := flags.Bool("json", false, "Print raw JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if flags.NArg() < 2 {
		return fmt.Errorf("usage: boids <agent-model> <input>")
	}

	model := flags.Arg(0)
	input := strings.Join(flags.Args()[1:], " ")
	client := boids.NewClient(*apiKey, boids.WithBaseURL(*baseURL))
	request := boids.ResponseRequest{Model: model, Input: input, Stream: *stream && !*noStream}
	return printResponse(context.Background(), client, request, *jsonOutput)
}

func ask(args []string) error {
	flags := flag.NewFlagSet("ask", flag.ContinueOnError)
	model := flags.String("model", os.Getenv("BOIDS_MODEL"), "Boids model")
	apiKey := flags.String("api-key", os.Getenv("BOIDS_API_KEY"), "Boids API key")
	baseURL := flags.String("base-url", boids.DefaultBaseURL, "Boids API base URL")
	stream := flags.Bool("stream", true, "Stream response events")
	noStream := flags.Bool("no-stream", false, "Disable response streaming")
	jsonOutput := flags.Bool("json", false, "Print raw JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *model == "" {
		return fmt.Errorf("missing -model or BOIDS_MODEL")
	}

	input := strings.Join(flags.Args(), " ")
	if input == "" {
		return fmt.Errorf("missing input text")
	}

	client := boids.NewClient(*apiKey, boids.WithBaseURL(*baseURL))
	request := boids.ResponseRequest{Model: *model, Input: input, Stream: *stream && !*noStream}
	return printResponse(context.Background(), client, request, *jsonOutput)
}

func createResponse(args []string) error {
	flags := flag.NewFlagSet("responses create", flag.ContinueOnError)
	model := flags.String("model", os.Getenv("BOIDS_MODEL"), "Boids model")
	input := flags.String("input", "", "Input text")
	apiKey := flags.String("api-key", os.Getenv("BOIDS_API_KEY"), "Boids API key")
	baseURL := flags.String("base-url", boids.DefaultBaseURL, "Boids API base URL")
	stream := flags.Bool("stream", false, "Stream response events")
	jsonOutput := flags.Bool("json", false, "Print raw JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *model == "" {
		return fmt.Errorf("missing -model or BOIDS_MODEL")
	}

	inputText := *input
	if inputText == "" {
		inputText = strings.Join(flags.Args(), " ")
	}
	if inputText == "" {
		return fmt.Errorf("missing -input or input text")
	}

	client := boids.NewClient(*apiKey, boids.WithBaseURL(*baseURL))
	request := boids.ResponseRequest{Model: *model, Input: inputText, Stream: *stream}
	return printResponse(context.Background(), client, request, *jsonOutput)
}

func printResponse(ctx context.Context, client *boids.Client, request boids.ResponseRequest, jsonOutput bool) error {
	if request.Stream {
		stream, err := client.StreamResponse(ctx, request)
		if err != nil {
			return err
		}
		defer stream.Close()

		wroteText := false
		fallbackText := ""
		for stream.Next() {
			event := stream.Event()
			if jsonOutput {
				if err := printJSON(event); err != nil {
					return err
				}
				continue
			}

			delta := extractDelta(event.Data)
			if delta != "" {
				fmt.Print(delta)
				wroteText = true
				continue
			}

			if strings.HasSuffix(event.Event, ".completed") {
				fallbackText = extractText(event.Data)
			}
		}
		if wroteText {
			fmt.Println()
		} else if fallbackText != "" {
			fmt.Println(fallbackText)
		}
		return stream.Err()
	}

	response, err := client.CreateResponse(ctx, request)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(response)
	}

	text := extractText(response)
	if text == "" {
		return printJSON(response)
	}
	fmt.Println(text)
	return nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func extractText(value any) string {
	switch item := value.(type) {
	case string:
		return item
	case []any:
		var builder strings.Builder
		for _, child := range item {
			builder.WriteString(extractText(child))
		}
		return builder.String()
	case map[string]any:
		for _, key := range []string{"delta", "text", "output_text"} {
			if text, ok := item[key].(string); ok {
				return text
			}
		}
		for _, key := range []string{"content", "output", "message"} {
			if text := extractText(item[key]); text != "" {
				return text
			}
		}
	}
	return ""
}

func extractDelta(value any) string {
	if item, ok := value.(map[string]any); ok {
		if delta, ok := item["delta"].(string); ok {
			return delta
		}
	}
	return ""
}

func usage() {
	fmt.Println(`Usage:
  boids <agent-model> <input>
  boids ask -model <model> <input>
  boids responses create -model <model> -input <input> [-stream]

Environment:
  BOIDS_API_KEY   Required API key
  BOIDS_MODEL     Optional default model`)
}
