# boids-sdk

Python SDK and CLI for the Boids API.

## Install

```bash
pip install boids-sdk
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

```python
from boids import BoidsClient

client = BoidsClient()

response = client.responses.create(
    model="agent:@iris-wei-org/my-doppelganger",
    input="Introduce yourself in one sentence.",
)
print(response)
```

Streaming:

```python
for event in client.responses.create(
    model="agent:@iris-wei-org/my-doppelganger",
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
