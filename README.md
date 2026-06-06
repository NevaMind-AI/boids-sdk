# Boids SDK

Python, JavaScript, and Go SDKs plus a `boids` CLI for the Boids API.

The SDKs are intentionally small. They handle the base URL, bearer auth, JSON
encoding, API errors, and server-sent event streaming while keeping request
bodies flexible for future Boids parameters.

## Install

CLI:

```bash
curl -fsSL https://raw.githubusercontent.com/NevaMind-AI/boids-sdk/main/install.sh | bash
```

Python:

```bash
pip install boids-sdk
```

JavaScript:

```bash
npm install boids-sdk
```

With a local npm install, run the CLI through `npx boids ...` or from npm
scripts. For a global CLI install, use the curl installer above.

Go:

```bash
go get github.com/NevaMind-AI/boids-sdk/go
```

Set your API key:

```bash
export BOIDS_API_KEY="..."
```

Use environment variables in production rather than hard-coding API keys.

## API Example

```bash
# Get your API key in Settings. Use environment variables in production.
curl "https://api.boids.so/v1/responses" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BOIDS_API_KEY" \
  -d '{
  "model": "agent:@boids-team/jarvis",
  "input": "Introduce yourself in one sentence.",
  "stream": true
}'
```

## CLI

Install the CLI with the bash installer, then set `BOIDS_API_KEY`:

```bash
curl -fsSL https://raw.githubusercontent.com/NevaMind-AI/boids-sdk/main/install.sh | bash
export BOIDS_API_KEY="..."
```

The installer tries `npm install -g boids-sdk` first, then falls back to
`pipx install boids-sdk` or `pip install --user boids-sdk`. To force a method:

```bash
curl -fsSL https://raw.githubusercontent.com/NevaMind-AI/boids-sdk/main/install.sh | BOIDS_INSTALL_METHOD=pipx bash
```

Call a known agent directly:

```bash
boids agent:@boids-team/jarvis "Introduce yourself in one sentence."
```

Search for agents in the market:

```bash
boids search "global launch growth agent" --limit 5
```

Find the best matching agent and run your prompt end to end:

```bash
boids run "Create a launch plan for a developer tool."
boids run "Write an SEO plan" --search-query "SEO growth agent"
```

`boids run` calls `/v1/market/search`, picks the first returned agent as the
best match, then sends your prompt to `/v1/responses`.

More explicit commands are also available:

```bash
boids ask --model agent:@boids-team/jarvis "Introduce yourself."
boids responses create --model agent:@boids-team/jarvis --input "Introduce yourself." --stream
boids chat --model agent:@boids-team/jarvis --input "Introduce yourself."
```

Continue a conversation by passing the previous response id:

```bash
boids agent:@boids-team/jarvis "Remember my name is Ada." --show-response-id
boids agent:@boids-team/jarvis "What is my name?" --prev resp_...
```

The CLI sends `previous_response_id` to `/v1/responses`. In stream mode,
`--show-response-id` prints the completed response id to stderr so stdout stays
clean for the assistant text.

You can set a default model:

```bash
export BOIDS_MODEL="agent:@boids-team/jarvis"
boids ask "Introduce yourself in one sentence."
```

## Python

```python
from boids import BoidsClient

client = BoidsClient()

response = client.responses.create(
    model="agent:@boids-team/jarvis",
    input="Introduce yourself in one sentence.",
)
print(response)
```

Chat complete:

```python
response = client.chat.complete(
    model="agent:@boids-team/jarvis",
    messages=[{"role": "user", "content": "Introduce yourself in one sentence."}],
)
print(response)
```

Streaming:

```python
for event in client.responses.create(
    model="agent:@boids-team/jarvis",
    input="Introduce yourself in one sentence.",
    stream=True,
):
    print(event.data)
```

Market search:

```python
agents = client.market.search(query="global launch growth agent", limit=5)
print(agents["data"]["items"][0]["model_name"])
```

Conversation context:

```python
first = client.responses.create(
    model="agent:@boids-team/jarvis",
    input="Remember my name is Ada.",
)

second = client.responses.create(
    model="agent:@boids-team/jarvis",
    input="What is my name?",
    previous_response_id=first["id"],
)
```

## JavaScript

```js
import { Boids } from "boids-sdk";

const client = new Boids();

const response = await client.responses.create({
  model: "agent:@boids-team/jarvis",
  input: "Introduce yourself in one sentence.",
});
console.log(response);
```

Chat complete:

```js
const response = await client.chat.complete({
  model: "agent:@boids-team/jarvis",
  messages: [{ role: "user", content: "Introduce yourself in one sentence." }],
});
console.log(response);
```

Streaming:

```js
for await (const event of client.responses.create({
  model: "agent:@boids-team/jarvis",
  input: "Introduce yourself in one sentence.",
  stream: true,
})) {
  console.log(event.data);
}
```

Market search:

```js
const agents = await client.market.search({
  query: "global launch growth agent",
  limit: 5,
});
console.log(agents.data.items[0].model_name);
```

Conversation context:

```js
const first = await client.responses.create({
  model: "agent:@boids-team/jarvis",
  input: "Remember my name is Ada.",
});

const second = await client.responses.create({
  model: "agent:@boids-team/jarvis",
  input: "What is my name?",
  previous_response_id: first.id,
});
```

## Go

```go
package main

import (
	"context"
	"fmt"
	"log"

	boids "github.com/NevaMind-AI/boids-sdk/go"
)

func main() {
	client := boids.NewClient("")

	response, err := client.CreateResponse(context.Background(), boids.ResponseRequest{
		Model: "agent:@boids-team/jarvis",
		Input: "Introduce yourself in one sentence.",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v\n", response)
}
```

Chat complete:

```go
response, err := client.CompleteChat(context.Background(), boids.ChatCompleteRequest{
	Model: "agent:@boids-team/jarvis",
	Messages: []map[string]string{
		{"role": "user", "content": "Introduce yourself in one sentence."},
	},
})
```

Conversation context:

```go
first, err := client.CreateResponse(context.Background(), boids.ResponseRequest{
	Model: "agent:@boids-team/jarvis",
	Input: "Remember my name is Ada.",
})
if err != nil {
	log.Fatal(err)
}

firstMap := first.(map[string]any)
second, err := client.CreateResponse(context.Background(), boids.ResponseRequest{
	Model:              "agent:@boids-team/jarvis",
	Input:              "What is my name?",
	PreviousResponseID: firstMap["id"].(string),
})
```

Run the Go CLI locally:

```bash
cd go
go run ./cmd/boids agent:@boids-team/jarvis "Introduce yourself in one sentence."
go run ./cmd/boids search "global launch growth agent" -limit 5
go run ./cmd/boids run "Create a launch plan for a developer tool."
```

## Configuration

- `BOIDS_API_KEY`: required API key.
- `BOIDS_MODEL`: optional default model for CLI commands.
- `https://api.boids.so/v1`: default API base URL.

## Repository Layout

- `python/`: Python SDK and CLI package published as `boids-sdk`.
- `js/`: JavaScript SDK and CLI package published as `boids-sdk`.
- `go/`: Go SDK plus optional `cmd/boids` CLI.
