# boids-sdk

Python SDK and CLI for the Boids API.

## Install

Install the SDK from PyPI:

```bash
pip install boids-sdk
```

Then set your API key:

```bash
export BOIDS_API_KEY="..."
```

Use environment variables in production rather than hard-coding API keys.

To install the standalone `boids` CLI globally instead, use the bash installer
(it tries npm first, then falls back to pipx or pip):

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
