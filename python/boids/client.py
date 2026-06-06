from __future__ import annotations

import json
import os
from dataclasses import dataclass
from typing import Any, Dict, Iterator, Mapping, Optional
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

DEFAULT_BASE_URL = "https://api.boids.so/v1"
USER_AGENT = "boids-python/0.1.0"


class BoidsError(Exception):
    """Base class for Boids SDK errors."""


class BoidsConnectionError(BoidsError):
    """Raised when the client cannot reach the Boids API."""


class BoidsAPIError(BoidsError):
    """Raised when the Boids API returns a non-2xx response."""

    def __init__(self, status_code: int, body: str):
        self.status_code = status_code
        self.body = body
        super().__init__(f"Boids API error {status_code}: {body}")


@dataclass(frozen=True)
class ResponseEvent:
    event: Optional[str]
    data: Any
    raw: str

    def to_dict(self) -> Dict[str, Any]:
        return {"event": self.event, "data": self.data, "raw": self.raw}


class ResponsesResource:
    def __init__(self, client: "BoidsClient"):
        self._client = client

    def create(
        self,
        *,
        model: str,
        input: Any,
        stream: bool = False,
        **params: Any,
    ) -> Any:
        body = dict(params)
        body["model"] = model
        body["input"] = input
        body["stream"] = stream
        return self._client._post("/responses", body, stream=stream)


class MarketResource:
    def __init__(self, client: "BoidsClient"):
        self._client = client

    def search(self, *, query: str, limit: int = 5, **params: Any) -> Any:
        body = dict(params)
        body["query"] = query
        body["limit"] = limit
        return self._client._post("/market/search", body, stream=False)


class BoidsClient:
    def __init__(
        self,
        api_key: Optional[str] = None,
        *,
        base_url: str = DEFAULT_BASE_URL,
        timeout: Optional[float] = 60,
        headers: Optional[Mapping[str, str]] = None,
    ):
        self.api_key = api_key or os.getenv("BOIDS_API_KEY")
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.headers = dict(headers or {})
        self.responses = ResponsesResource(self)
        self.market = MarketResource(self)

    def _post(self, path: str, body: Mapping[str, Any], *, stream: bool) -> Any:
        response = self._open(path, body)
        if stream:
            return _iter_sse_events(response)
        return _read_json_response(response)

    def _open(self, path: str, body: Mapping[str, Any]) -> Any:
        if not self.api_key:
            raise BoidsError("Missing BOIDS_API_KEY or api_key.")

        payload = json.dumps(_without_none(dict(body))).encode("utf-8")
        request = Request(
            f"{self.base_url}{path}",
            data=payload,
            method="POST",
            headers={
                "Authorization": f"Bearer {self.api_key}",
                "Content-Type": "application/json",
                "Accept": "text/event-stream, application/json",
                "User-Agent": USER_AGENT,
                **self.headers,
            },
        )

        try:
            kwargs: Dict[str, Any] = {}
            if self.timeout is not None:
                kwargs["timeout"] = self.timeout
            return urlopen(request, **kwargs)
        except HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            raise BoidsAPIError(exc.code, body) from exc
        except URLError as exc:
            raise BoidsConnectionError(str(exc.reason)) from exc


def _without_none(value: Dict[str, Any]) -> Dict[str, Any]:
    return {key: item for key, item in value.items() if item is not None}


def _read_json_response(response: Any) -> Any:
    try:
        raw = response.read().decode("utf-8")
    finally:
        response.close()

    if not raw:
        return None

    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return raw


def _iter_sse_events(response: Any) -> Iterator[ResponseEvent]:
    event_name: Optional[str] = None
    data_lines = []
    raw_lines = []

    try:
        for raw_line in response:
            line = raw_line.decode("utf-8", errors="replace").rstrip("\r\n")

            if line == "":
                event = _make_event(event_name, data_lines, raw_lines)
                event_name = None
                data_lines = []
                raw_lines = []
                if event is None:
                    continue
                if event.raw == "[DONE]":
                    break
                yield event
                continue

            raw_lines.append(line)
            if line.startswith(":"):
                continue

            field, _, value = line.partition(":")
            if value.startswith(" "):
                value = value[1:]

            if field == "event":
                event_name = value
            elif field == "data":
                data_lines.append(value)

        event = _make_event(event_name, data_lines, raw_lines)
        if event is not None and event.raw != "[DONE]":
            yield event
    finally:
        response.close()


def _make_event(
    event_name: Optional[str],
    data_lines: list[str],
    raw_lines: list[str],
) -> Optional[ResponseEvent]:
    if not data_lines:
        return None

    raw_data = "\n".join(data_lines)
    if raw_data == "[DONE]":
        return ResponseEvent(event=event_name, data=raw_data, raw=raw_data)

    try:
        data = json.loads(raw_data)
    except json.JSONDecodeError:
        data = raw_data

    return ResponseEvent(event=event_name, data=data, raw="\n".join(raw_lines))
