# Boids SDK

Lightweight SDKs and CLI wrappers for the Boids Responses API.

The SDKs are intentionally small: they handle the base URL, bearer auth,
JSON encoding, API errors, and server-sent event streaming while keeping the
request body flexible for future Boids parameters.

```bash
export BOIDS_API_KEY="..."

curl "https://api.boids.so/v1/responses" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BOIDS_API_KEY" \
  -d '{
    "model": "agent:@iris-wei-org/my-doppelganger",
    "input": "Introduce yourself in one sentence.",
    "stream": true
  }'
```

## Layout

- `python/`: Python SDK plus a `boids` CLI.
- `js/`: JavaScript SDK plus a `boids` CLI for Node.js.
- `go/`: Go SDK plus an optional `cmd/boids` CLI.

## Python

Install locally:

```bash
cd python
python -m pip install -e .
```

Use the SDK:

```python
from boids import BoidsClient

client = BoidsClient()

response = client.responses.create(
    model="agent:@iris-wei-org/my-doppelganger",
    input="Introduce yourself in one sentence.",
)
print(response)

for event in client.responses.create(
    model="agent:@iris-wei-org/my-doppelganger",
    input="Introduce yourself in one sentence.",
    stream=True,
):
    print(event.data)
```

Use the CLI:

```bash
boids agent:@iris-wei-org/my-doppelganger "Introduce yourself in one sentence."
boids search "global launch growth agent" --limit 5
boids run "Create a launch plan for a developer tool."
boids ask --model agent:@iris-wei-org/my-doppelganger "Introduce yourself in one sentence."
boids responses create --model agent:@iris-wei-org/my-doppelganger --input "Introduce yourself in one sentence." --stream
```

`boids run` searches `/v1/market/search`, picks the first returned agent as the
best match, then sends the prompt to `/v1/responses`. Use `--search-query` when
you want one query for agent discovery and another prompt for execution.

You can also set a default model:

```bash
export BOIDS_MODEL="agent:@iris-wei-org/my-doppelganger"
boids ask "Introduce yourself in one sentence."
```

## JavaScript

Install locally:

```bash
cd js
npm install
npm link
```

Use the SDK:

```js
import { Boids } from "@boids/sdk";

const client = new Boids();

const response = await client.responses.create({
  model: "agent:@iris-wei-org/my-doppelganger",
  input: "Introduce yourself in one sentence.",
});
console.log(response);

for await (const event of client.responses.create({
  model: "agent:@iris-wei-org/my-doppelganger",
  input: "Introduce yourself in one sentence.",
  stream: true,
})) {
  console.log(event.data);
}
```

Use the CLI:

```bash
boids agent:@iris-wei-org/my-doppelganger "Introduce yourself in one sentence."
boids search "global launch growth agent" --limit 5
boids run "Create a launch plan for a developer tool."
boids ask --model agent:@iris-wei-org/my-doppelganger "Introduce yourself in one sentence."
boids responses create --model agent:@iris-wei-org/my-doppelganger --input "Introduce yourself in one sentence." --stream
```

## Go

Use the SDK:

```go
package main

import (
	"context"
	"fmt"
	"log"

	boids "github.com/boids/boids-go"
)

func main() {
	client := boids.NewClient("")

	response, err := client.CreateResponse(context.Background(), boids.ResponseRequest{
		Model: "agent:@iris-wei-org/my-doppelganger",
		Input: "Introduce yourself in one sentence.",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v\n", response)
}
```

Run the Go CLI locally:

```bash
cd go
go run ./cmd/boids agent:@iris-wei-org/my-doppelganger "Introduce yourself in one sentence."
go run ./cmd/boids search "global launch growth agent" -limit 5
go run ./cmd/boids run "Create a launch plan for a developer tool."
go run ./cmd/boids ask -model agent:@iris-wei-org/my-doppelganger "Introduce yourself in one sentence."
```

## Configuration

All SDKs and CLIs use:

- `BOIDS_API_KEY`: required API key.
- `BOIDS_MODEL`: optional default model for CLI commands.
- `https://api.boids.so/v1`: default API base URL.

Use environment variables in production rather than hard-coding API keys.
