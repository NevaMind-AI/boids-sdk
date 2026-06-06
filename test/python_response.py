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

        if self.path != "/responses":
            self.send_error(404)
            return
        if self.headers.get("Authorization") != "Bearer test-key":
            self.send_error(401)
            return

        if body.get("stream"):
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            self.wfile.write(b'event: response.output_text.delta\ndata: {"delta": "Hello"}\n\n')
            self.wfile.write(b'event: response.completed\ndata: {"id": "resp_123", "output_text": "Hello"}\n\n')
            self.wfile.write(b"data: [DONE]\n\n")
            return

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(
            json.dumps(
                {
                    "id": "resp_123",
                    "output_text": "Hello",
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
        base_url = f"http://127.0.0.1:{server.server_port}"
        client = BoidsClient(api_key="test-key", base_url=base_url)

        response = client.responses.create(
            model="agent:test",
            input="Say hello",
            temperature=0.2,
        )
        print("[sdk create]", json.dumps(response, ensure_ascii=False))
        assert response["id"] == "resp_123"
        assert response["output_text"] == "Hello"

        events = list(
            client.responses.create(
                model="agent:test",
                input="Say hello",
                stream=True,
            )
        )
        print("[sdk stream]", json.dumps([event.to_dict() for event in events], ensure_ascii=False))
        assert [event.event for event in events] == [
            "response.output_text.delta",
            "response.completed",
        ]
        assert events[0].data["delta"] == "Hello"
        assert events[1].data["id"] == "resp_123"

        stdout = StringIO()
        with redirect_stdout(stdout):
            exit_code = cli_main(
                [
                    "--api-key",
                    "test-key",
                    "--base-url",
                    base_url,
                    "responses",
                    "create",
                    "--model",
                    "agent:test",
                    "--input",
                    "Say hello",
                    "--json",
                ]
            )
        assert exit_code == 0
        create_response = json.loads(stdout.getvalue())
        print("[cli responses create]", json.dumps(create_response, ensure_ascii=False))
        assert create_response["id"] == "resp_123"

        stdout = StringIO()
        with redirect_stdout(stdout):
            exit_code = cli_main(
                [
                    "--api-key",
                    "test-key",
                    "--base-url",
                    base_url,
                    "ask",
                    "--model",
                    "agent:test",
                    "--no-stream",
                    "--json",
                    "Say hello",
                ]
            )
        assert exit_code == 0
        ask_response = json.loads(stdout.getvalue())
        print("[cli ask]", json.dumps(ask_response, ensure_ascii=False))
        assert ask_response["id"] == "resp_123"

        records = server.records  # type: ignore[attr-defined]
        assert [record["path"] for record in records] == ["/responses"] * 4
        assert records[0]["authorization"] == "Bearer test-key"
        assert records[0]["body"]["model"] == "agent:test"
        assert records[0]["body"]["input"] == "Say hello"
        assert records[0]["body"]["stream"] is False
        assert records[0]["body"]["temperature"] == 0.2
        assert records[1]["body"]["stream"] is True
        assert records[2]["body"]["input"] == "Say hello"
        assert records[2]["body"]["stream"] is False
        assert records[3]["body"]["input"] == "Say hello"
        assert records[3]["body"]["stream"] is False

        print("python responses test passed")
        return 0
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


if __name__ == "__main__":
    raise SystemExit(main())
