package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	boids "github.com/NevaMind-AI/boids-sdk/go"
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
	case "search":
		return searchMarket(args[1:])
	case "run", "auto":
		return runAuto(args[1:])
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

func searchMarket(args []string) error {
	flags := flag.NewFlagSet("search", flag.ContinueOnError)
	apiKey := flags.String("api-key", os.Getenv("BOIDS_API_KEY"), "Boids API key")
	baseURL := flags.String("base-url", boids.DefaultBaseURL, "Boids API base URL")
	limit := flags.Int("limit", 5, "Maximum number of agents")
	jsonOutput := flags.Bool("json", false, "Print raw JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	query := strings.Join(flags.Args(), " ")
	if query == "" {
		return fmt.Errorf("missing search query")
	}

	client := boids.NewClient(*apiKey, boids.WithBaseURL(*baseURL))
	result, err := client.SearchMarket(context.Background(), boids.MarketSearchRequest{
		Query: query,
		Limit: *limit,
	})
	if err != nil {
		return err
	}

	if *jsonOutput {
		return printJSON(result)
	}

	printSearchResult(result)
	return nil
}

func runAuto(args []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	apiKey := flags.String("api-key", os.Getenv("BOIDS_API_KEY"), "Boids API key")
	baseURL := flags.String("base-url", boids.DefaultBaseURL, "Boids API base URL")
	searchQuery := flags.String("search-query", "", "Use a different query for market search")
	limit := flags.Int("limit", 1, "Maximum number of agents to search")
	stream := flags.Bool("stream", true, "Stream response events")
	noStream := flags.Bool("no-stream", false, "Disable response streaming")
	jsonOutput := flags.Bool("json", false, "Print raw JSON")
	quietAgent := flags.Bool("quiet-agent", false, "Do not print selected agent")
	if err := flags.Parse(args); err != nil {
		return err
	}

	input := strings.Join(flags.Args(), " ")
	if input == "" {
		return fmt.Errorf("missing input text")
	}

	query := *searchQuery
	if query == "" {
		query = input
	}

	client := boids.NewClient(*apiKey, boids.WithBaseURL(*baseURL))
	item, model, err := selectBestAgent(context.Background(), client, query, *limit)
	if err != nil {
		return err
	}

	if !*quietAgent && !*jsonOutput {
		fmt.Fprintf(os.Stderr, "Selected agent: %s (%s)\n", marketTitle(item, model), model)
	}

	request := boids.ResponseRequest{Model: model, Input: input, Stream: *stream && !*noStream}
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

func selectBestAgent(ctx context.Context, client *boids.Client, query string, limit int) (map[string]any, string, error) {
	result, err := client.SearchMarket(ctx, boids.MarketSearchRequest{Query: query, Limit: limit})
	if err != nil {
		return nil, "", err
	}

	items := marketItems(result)
	if len(items) == 0 {
		return nil, "", fmt.Errorf("no agents found for: %s", query)
	}

	model := agentModel(items[0])
	if model == "" {
		return nil, "", fmt.Errorf("best market result did not include a usable model")
	}

	return items[0], model, nil
}

func printSearchResult(result any) {
	items := marketItems(result)
	if len(items) == 0 {
		fmt.Println("No agents found.")
		return
	}

	for index, item := range items {
		model := agentModel(item)
		if model == "" {
			model = "unknown"
		}
		fmt.Printf("%d. %s\n", index+1, marketTitle(item, model))
		fmt.Printf("   model: %s\n", model)
		if description, ok := item["description"].(string); ok && description != "" {
			fmt.Printf("   %s\n", description)
		}
	}
}

func marketItems(result any) []map[string]any {
	root, ok := result.(map[string]any)
	if !ok {
		return nil
	}
	data, ok := root["data"].(map[string]any)
	if !ok {
		return nil
	}
	rawItems, ok := data["items"].([]any)
	if !ok {
		return nil
	}

	items := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		if ok {
			items = append(items, item)
		}
	}
	return items
}

func agentModel(item map[string]any) string {
	if modelName, ok := item["model_name"].(string); ok && strings.HasPrefix(modelName, "agent:") {
		return modelName
	}

	if agentID, ok := item["agent_id"].(string); ok && agentID != "" {
		return "agent:" + agentID
	}
	if id, ok := item["id"].(string); ok && id != "" {
		return "agent:" + id
	}
	if modelName, ok := item["model_name"].(string); ok && modelName != "" {
		return modelName
	}

	return ""
}

func marketTitle(item map[string]any, fallback string) string {
	if title, ok := item["title"].(string); ok && title != "" {
		return title
	}
	if id, ok := item["id"].(string); ok && id != "" {
		return id
	}
	return fallback
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
  boids search <query> [-limit 5]
  boids run <input> [-search-query <query>]
  boids ask -model <model> <input>
  boids responses create -model <model> -input <input> [-stream]

Environment:
  BOIDS_API_KEY   Required API key
  BOIDS_MODEL     Optional default model`)
}
