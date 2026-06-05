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

    responses = subcommands.add_parser("responses", help="Work with responses.")
    response_commands = responses.add_subparsers(dest="response_command", required=True)

    create = response_commands.add_parser("create", help="Create a response.")
    create.add_argument("-m", "--model", default=os.getenv("BOIDS_MODEL"))
    create.add_argument("-i", "--input")
    create.add_argument("--json", action="store_true", help="Print raw JSON output.")
    create.add_argument("--stream", action="store_true")

    args = parser.parse_args(argv)

    client = BoidsClient(api_key=args.api_key, base_url=args.base_url)

    if args.command == "ask":
        model = _require_model(args.model)
        input_text = " ".join(args.input)
        result = client.responses.create(
            model=model,
            input=input_text,
            stream=args.stream,
        )
        _print_result(result, as_json=args.json)
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
        )
        _print_result(result, as_json=args.json)
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
    parser.add_argument("model", help="Boids agent model, for example agent:@org/name.")
    parser.add_argument("input", nargs="+", help="Prompt text.")

    args = parser.parse_args(argv)
    client = BoidsClient(api_key=args.api_key, base_url=args.base_url)
    result = client.responses.create(
        model=args.model,
        input=" ".join(args.input),
        stream=args.stream,
    )
    _print_result(result, as_json=args.json)
    return 0


def _looks_like_shortcut(argv: list[str]) -> bool:
    first = _first_positional(argv)
    return first is not None and first not in {"ask", "responses"}


def _first_positional(argv: list[str]) -> Optional[str]:
    skip_next = False
    options_with_values = {"--api-key", "--base-url"}

    for item in argv:
        if skip_next:
            skip_next = False
            continue

        if item in options_with_values:
            skip_next = True
            continue

        if item.startswith("--api-key=") or item.startswith("--base-url="):
            continue

        if item.startswith("-"):
            continue

        return item

    return None


def _require_model(model: Optional[str]) -> str:
    if model:
        return model
    raise SystemExit("Missing --model or BOIDS_MODEL.")


def _print_result(result: Any, *, as_json: bool) -> None:
    if isinstance(result, Iterable) and not isinstance(result, (dict, list, str, bytes)):
        _print_stream(result, as_json=as_json)
        return

    if as_json:
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return

    text = extract_text(result)
    if text is None:
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        print(text)


def _print_stream(events: Iterable[ResponseEvent], *, as_json: bool) -> None:
    wrote_text = False
    fallback_text: Optional[str] = None

    for event in events:
        if as_json:
            print(json.dumps(event.to_dict(), ensure_ascii=False))
            continue

        delta = extract_delta(event.data)
        if delta is not None:
            _write(delta)
            sys.stdout.flush()
            wrote_text = True
            continue

        if event.event and event.event.endswith(".completed"):
            fallback_text = extract_text(event.data)

    if wrote_text:
        _write("\n")
    elif fallback_text:
        print(fallback_text)


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
