# boids-sdk

JavaScript SDK and CLI for the Boids API.

## Install

Install the SDK from npm:

```bash
npm install boids-sdk
```

Then set your API key:

```bash
export BOIDS_API_KEY="..."
```

Use environment variables in production rather than hard-coding API keys. With a
local install, you can also run the bundled CLI through `npx boids ...`.

To install the standalone `boids` CLI globally instead, use the bash installer
(it tries `npm install -g boids-sdk` first, then falls back to pipx or pip):

```bash
curl -fsSL https://raw.githubusercontent.com/NevaMind-AI/boids-sdk/main/install.sh | bash
```

## CLI

```bash
boids agent:@boids-team/jarvis "Introduce yourself in one sentence."
boids search "global launch growth agent" --limit 5
boids run "Create a launch plan for a developer tool."
boids agent:@boids-team/jarvis "Remember my name is Ada." --show-response-id
boids agent:@boids-team/jarvis "What is my name?" --prev resp_...
```

`boids run` searches `/v1/market/search`, selects the first returned agent, and
then sends your prompt to `/v1/responses`.

## SDK

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
