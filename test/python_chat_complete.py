#!/usr/bin/env python3
from __future__ import annotations

import json
import sys
from contextlib import redirect_stdout
from http.server import BaseHTTPRequestHandler, HTTPServer
from io import StringIO
from pathlib import Path
from threading import Thread
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "python"))

from boids import BoidsClient  # noqa: E402
from boids.cli import main as cli_main  # noqa: E402


class MockBoidsHandler(BaseHTTPRequestHandler):
    def do_POST(self) -> None:
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length).decode("utf-8"))
        self.server.records.append(  # type: ignore[attr-defined]
            {
                "path": self.path,
                "authorization": self.headers.get("Authorization"),
                "body": body,
            }
        )

        if self.path != "/chat/complete":
            self.send_error(404)
            return
        if self.headers.get("Authorization") != "Bearer test-key":
            self.send_error(401)
            return

        if body.get("stream"):
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            self.wfile.write(b'event: chat.delta\ndata: {"delta": "Hello"}\n\n')
            self.wfile.write(b'event: chat.completed\ndata: {"id": "chat_123", "output_text": "Hello"}\n\n')
            self.wfile.write(b"data: [DONE]\n\n")
            return

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(
            json.dumps(
                {
                    "id": "chat_123",
                    "message": {"role": "assistant", "content": "Hello"},
                }
            ).encode("utf-8")
        )

    def log_message(self, format: str, *args: Any) -> None:
        return


def main() -> int:
    server = HTTPServer(("127.0.0.1", 0), MockBoidsHandler)
    server.records = []  # type: ignore[attr-defined]
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()

    try:
        client = BoidsClient(
            api_key="test-key",
            base_url=f"http://127.0.0.1:{server.server_port}",
        )
        messages = [{"role": "user", "content": "Say hello"}]

        response = client.chat.complete(
            model="agent:test",
            messages=messages,
            temperature=0.2,
        )
        assert response["id"] == "chat_123"
        assert response["message"]["content"] == "Hello"

        events = list(
            client.chat.complete(
                model="agent:test",
                messages=messages,
                stream=True,
            )
        )
        assert [event.event for event in events] == ["chat.delta", "chat.completed"]
        assert events[0].data["delta"] == "Hello"
        assert events[1].data["id"] == "chat_123"

        stdout = StringIO()
        with redirect_stdout(stdout):
            exit_code = cli_main(
                [
                    "--api-key",
                    "test-key",
                    "--base-url",
                    f"http://127.0.0.1:{server.server_port}",
                    "chat",
                    "--model",
                    "agent:test",
                    "--input",
                    "Say hello",
                    "--json",
                ]
            )
        assert exit_code == 0
        cli_response = json.loads(stdout.getvalue())
        assert cli_response["id"] == "chat_123"

        records = server.records  # type: ignore[attr-defined]
        assert [record["path"] for record in records] == [
            "/chat/complete",
            "/chat/complete",
            "/chat/complete",
        ]
        assert records[0]["authorization"] == "Bearer test-key"
        assert records[0]["body"]["model"] == "agent:test"
        assert records[0]["body"]["messages"] == messages
        assert records[0]["body"]["stream"] is False
        assert records[0]["body"]["temperature"] == 0.2
        assert records[1]["body"]["stream"] is True
        assert records[2]["body"]["messages"] == [{"role": "user", "content": "Say hello"}]
        assert records[2]["body"]["stream"] is False

        print("python chat/complete test passed")
        return 0
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


if __name__ == "__main__":
    raise SystemExit(main())
