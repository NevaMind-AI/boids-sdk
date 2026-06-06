# boids-sdk

JavaScript SDK and CLI for the Boids API.

## Install

```bash
npm install boids-sdk
export BOIDS_API_KEY="..."
```

## CLI

```bash
boids agent:@iris-wei-org/my-doppelganger "Introduce yourself in one sentence."
boids search "global launch growth agent" --limit 5
boids run "Create a launch plan for a developer tool."
```

`boids run` searches `/v1/market/search`, selects the first returned agent, and
then sends your prompt to `/v1/responses`.

## SDK

```js
import { Boids } from "boids-sdk";

const client = new Boids();

const response = await client.responses.create({
  model: "agent:@iris-wei-org/my-doppelganger",
  input: "Introduce yourself in one sentence.",
});
console.log(response);
```

Streaming:

```js
for await (const event of client.responses.create({
  model: "agent:@iris-wei-org/my-doppelganger",
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
