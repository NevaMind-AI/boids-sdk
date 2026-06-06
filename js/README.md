# boids-sdk

JavaScript SDK and CLI for the Boids API.

## Install

```bash
npm install boids-sdk
export BOIDS_API_KEY="..."
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
