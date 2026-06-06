from __future__ import annotations

import argparse
import json
import os
import sys
from collections.abc import Iterable
from typing import Any, Optional

from .client import DEFAULT_BASE_URL, BoidsClient, ResponseEvent


def main(argv: Optional[list[str]] = None) -> int:
    _configure_stdio()
    argv = list(argv if argv is not None else sys.argv[1:])

    if _looks_like_shortcut(argv):
        return _run_shortcut(argv)

    parser = argparse.ArgumentParser(
        prog="boids",
        epilog='Shortcut: boids <agent-model> "your query"',
    )
    parser.add_argument("--api-key", default=os.getenv("BOIDS_API_KEY"))
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)

    subcommands = parser.add_subparsers(dest="command", required=True)

    ask = subcommands.add_parser("ask", help="Send one prompt to a Boids model.")
    ask.add_argument("input", nargs="+")
    ask.add_argument("-m", "--model", default=os.getenv("BOIDS_MODEL"))
    ask.add_argument("--json", action="store_true", help="Print raw JSON output.")
    ask.add_argument("--stream", dest="stream", action="store_true", default=True)
    ask.add_argument("--no-stream", dest="stream", action="store_false")
    ask.add_argument("--previous-response-id", "--prev", dest="previous_response_id")
    ask.add_argument("--show-response-id", action="store_true")

    search = subcommands.add_parser("search", help="Search Boids market agents.")
    search.add_argument("query", nargs="+")
    search.add_argument("--limit", type=int, default=5)
    search.add_argument("--json", action="store_true", help="Print raw JSON output.")

    run = subcommands.add_parser(
        "run",
        aliases=["auto"],
        help="Find the best matching agent and send a prompt to it.",
    )
    run.add_argument("input", nargs="+")
    run.add_argument("--search-query", help="Use a different query for market search.")
    run.add_argument("--limit", type=int, default=1)
    run.add_argument("--json", action="store_true", help="Print raw JSON output.")
    run.add_argument("--stream", dest="stream", action="store_true", default=True)
    run.add_argument("--no-stream", dest="stream", action="store_false")
    run.add_argument("--previous-response-id", "--prev", dest="previous_response_id")
    run.add_argument("--show-response-id", action="store_true")
    run.add_argument("--show-agent", dest="show_agent", action="store_true", default=True)
    run.add_argument("--quiet-agent", dest="show_agent", action="store_false")

    responses = subcommands.add_parser("responses", help="Work with responses.")
    response_commands = responses.add_subparsers(dest="response_command", required=True)

    create = response_commands.add_parser("create", help="Create a response.")
    create.add_argument("-m", "--model", default=os.getenv("BOIDS_MODEL"))
    create.add_argument("-i", "--input")
    create.add_argument("--json", action="store_true", help="Print raw JSON output.")
    create.add_argument("--stream", action="store_true")
    create.add_argument("--previous-response-id", "--prev", dest="previous_response_id")
    create.add_argument("--show-response-id", action="store_true")

    chat = subcommands.add_parser("chat", help="Create a chat completion.")
    chat.add_argument("-m", "--model", default=os.getenv("BOIDS_MODEL"))
    chat.add_argument("-i", "--input")
    chat.add_argument("--message", action="append", default=[], help="Add a role:content message.")
    chat.add_argument("--json", action="store_true", help="Print raw JSON output.")
    chat.add_argument("--stream", action="store_true")
    chat.add_argument("--show-response-id", action="store_true")
    chat.add_argument("input_args", nargs="*")

    args = parser.parse_args(argv)

    client = BoidsClient(api_key=args.api_key, base_url=args.base_url)

    if args.command == "ask":
        model = _require_model(args.model)
        input_text = " ".join(args.input)
        result = client.responses.create(
            model=model,
            input=input_text,
            stream=args.stream,
            previous_response_id=args.previous_response_id,
        )
        _print_result(result, as_json=args.json, show_response_id=args.show_response_id)
        return 0

    if args.command == "search":
        result = client.market.search(query=" ".join(args.query), limit=args.limit)
        _print_search_result(result, as_json=args.json)
        return 0

    if args.command in {"run", "auto"}:
        input_text = " ".join(args.input)
        result = _run_with_best_agent(
            client,
            input_text=input_text,
            search_query=args.search_query or input_text,
            limit=args.limit,
            stream=args.stream,
            previous_response_id=args.previous_response_id,
            show_agent=args.show_agent and not args.json,
        )
        _print_result(result, as_json=args.json, show_response_id=args.show_response_id)
        return 0

    if args.command == "responses" and args.response_command == "create":
        model = _require_model(args.model)
        input_text = args.input
        if input_text is None and not sys.stdin.isatty():
            input_text = sys.stdin.read()
        if not input_text:
            raise SystemExit("Missing --input or piped stdin.")
        result = client.responses.create(
            model=model,
            input=input_text,
            stream=args.stream,
            previous_response_id=args.previous_response_id,
        )
        _print_result(result, as_json=args.json, show_response_id=args.show_response_id)
        return 0

    if args.command == "chat":
        model = _require_model(args.model)
        result = client.chat.complete(
            model=model,
            messages=_chat_messages(args.input, args.input_args, args.message),
            stream=args.stream,
        )
        _print_result(result, as_json=args.json, show_response_id=args.show_response_id)
        return 0

    parser.error("Unknown command.")
    return 2


def _run_shortcut(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(
        prog="boids",
        description="Send one prompt to a Boids agent.",
    )
    parser.add_argument("--api-key", default=os.getenv("BOIDS_API_KEY"))
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)
    parser.add_argument("--json", action="store_true", help="Print raw JSON output.")
    parser.add_argument("--stream", dest="stream", action="store_true", default=True)
    parser.add_argument("--no-stream", dest="stream", action="store_false")
    parser.add_argument("--previous-response-id", "--prev", dest="previous_response_id")
    parser.add_argument("--show-response-id", action="store_true")
    parser.add_argument("model", help="Boids agent model, for example agent:@org/name.")
    parser.add_argument("input", nargs="+", help="Prompt text.")

    args = parser.parse_args(argv)
    client = BoidsClient(api_key=args.api_key, base_url=args.base_url)
    result = client.responses.create(
        model=args.model,
        input=" ".join(args.input),
        stream=args.stream,
        previous_response_id=args.previous_response_id,
    )
    _print_result(result, as_json=args.json, show_response_id=args.show_response_id)
    return 0


def _looks_like_shortcut(argv: list[str]) -> bool:
    first = _first_positional(argv)
    return first is not None and first not in {"ask", "responses", "chat", "search", "run", "auto"}


def _first_positional(argv: list[str]) -> Optional[str]:
    skip_next = False
    options_with_values = {"--api-key", "--base-url", "--previous-response-id", "--prev"}

    for item in argv:
        if skip_next:
            skip_next = False
            continue

        if item in options_with_values:
            skip_next = True
            continue

        if (
            item.startswith("--api-key=")
            or item.startswith("--base-url=")
            or item.startswith("--previous-response-id=")
            or item.startswith("--prev=")
        ):
            continue

        if item.startswith("-"):
            continue

        return item

    return None


def _require_model(model: Optional[str]) -> str:
    if model:
        return model
    raise SystemExit("Missing --model or BOIDS_MODEL.")


def _run_with_best_agent(
    client: BoidsClient,
    *,
    input_text: str,
    search_query: str,
    limit: int,
    stream: bool,
    previous_response_id: Optional[str],
    show_agent: bool,
) -> Any:
    search_result = client.market.search(query=search_query, limit=limit)
    item = _first_market_item(search_result)
    if item is None:
        raise SystemExit(f"No agents found for: {search_query}")

    model = _agent_model(item)
    if model is None:
        raise SystemExit("Best market result did not include a usable model.")

    if show_agent:
        title = item.get("title") or item.get("id") or model
        print(f"Selected agent: {title} ({model})", file=sys.stderr)

    return client.responses.create(
        model=model,
        input=input_text,
        stream=stream,
        previous_response_id=previous_response_id,
    )


def _chat_messages(
    input_text: Optional[str],
    input_args: list[str],
    raw_messages: list[str],
) -> list[dict[str, str]]:
    messages: list[dict[str, str]] = []
    for raw_message in raw_messages:
        role, separator, content = raw_message.partition(":")
        if not separator or not role or not content:
            raise SystemExit("--message must be formatted as role:content.")
        messages.append({"role": role, "content": content})

    if input_text is None and input_args:
        input_text = " ".join(input_args)
    if input_text is None and not sys.stdin.isatty():
        input_text = sys.stdin.read()
    if input_text:
        messages.append({"role": "user", "content": input_text})
    if not messages:
        raise SystemExit("Missing --input, --message, piped stdin, or input text.")
    return messages


def _print_search_result(result: Any, *, as_json: bool) -> None:
    if as_json:
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return

    items = _market_items(result)
    if not items:
        print("No agents found.")
        return

    for index, item in enumerate(items, start=1):
        title = item.get("title") or item.get("id") or "Untitled agent"
        model = _agent_model(item) or "unknown"
        description = item.get("description") or ""
        print(f"{index}. {title}")
        print(f"   model: {model}")
        if description:
            print(f"   {description}")


def _first_market_item(result: Any) -> Optional[dict[str, Any]]:
    items = _market_items(result)
    return items[0] if items else None


def _market_items(result: Any) -> list[dict[str, Any]]:
    if not isinstance(result, dict):
        return []
    data = result.get("data")
    if not isinstance(data, dict):
        return []
    items = data.get("items")
    if not isinstance(items, list):
        return []
    return [item for item in items if isinstance(item, dict)]


def _agent_model(item: dict[str, Any]) -> Optional[str]:
    model_name = item.get("model_name")
    if isinstance(model_name, str) and model_name.startswith("agent:"):
        return model_name

    agent_id = item.get("agent_id") or item.get("id")
    if isinstance(agent_id, str) and agent_id:
        return f"agent:{agent_id}"

    if isinstance(model_name, str) and model_name:
        return model_name

    return None


def _print_result(result: Any, *, as_json: bool, show_response_id: bool = False) -> None:
    if isinstance(result, Iterable) and not isinstance(result, (dict, list, str, bytes)):
        _print_stream(result, as_json=as_json, show_response_id=show_response_id)
        return

    if as_json:
        print(json.dumps(result, ensure_ascii=False, indent=2))
        if show_response_id:
            _print_response_id(result)
        return

    text = extract_text(result)
    if text is None:
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        print(text)

    if show_response_id:
        _print_response_id(result)


def _print_stream(
    events: Iterable[ResponseEvent],
    *,
    as_json: bool,
    show_response_id: bool,
) -> None:
    wrote_text = False
    fallback_text: Optional[str] = None
    response_id: Optional[str] = None

    for event in events:
        if event.event and event.event.endswith(".completed"):
            fallback_text = extract_text(event.data)
            response_id = extract_response_id(event.data)

        if as_json:
            print(json.dumps(event.to_dict(), ensure_ascii=False))
            continue

        delta = extract_delta(event.data)
        if delta is not None:
            _write(delta)
            sys.stdout.flush()
            wrote_text = True
            continue

    if wrote_text:
        _write("\n")
    elif fallback_text:
        print(fallback_text)

    if show_response_id and response_id:
        print(f"Response ID: {response_id}", file=sys.stderr)


def _print_response_id(value: Any) -> None:
    response_id = extract_response_id(value)
    if response_id:
        print(f"Response ID: {response_id}", file=sys.stderr)


def extract_response_id(value: Any) -> Optional[str]:
    if isinstance(value, dict):
        item = value.get("id")
        if isinstance(item, str):
            return item
    return None


def extract_delta(value: Any) -> Optional[str]:
    if isinstance(value, dict):
        item = value.get("delta")
        if isinstance(item, str):
            return item
    return None


def _configure_stdio() -> None:
    for stream in (sys.stdout, sys.stderr):
        reconfigure = getattr(stream, "reconfigure", None)
        if reconfigure is not None:
            try:
                reconfigure(encoding="utf-8", errors="replace")
            except TypeError:
                reconfigure(errors="replace")


def _write(text: str) -> None:
    try:
        sys.stdout.write(text)
    except UnicodeEncodeError:
        sys.stdout.write(text.encode(sys.stdout.encoding or "utf-8", errors="backslashreplace").decode(sys.stdout.encoding or "utf-8"))


def extract_text(value: Any) -> Optional[str]:
    if isinstance(value, str):
        return value

    if isinstance(value, dict):
        for key in ("delta", "text", "output_text"):
            item = value.get(key)
            if isinstance(item, str):
                return item

        for key in ("content", "output", "message"):
            item = value.get(key)
            text = extract_text(item)
            if text:
                return text

    if isinstance(value, list):
        parts = [part for item in value if (part := extract_text(item))]
        if parts:
            return "".join(parts)

    return None


if __name__ == "__main__":
    raise SystemExit(main())
