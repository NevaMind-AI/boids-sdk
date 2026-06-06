#!/usr/bin/env node
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createServer } from "node:http";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { Boids } from "../js/src/index.js";

const calls = [];
const client = new Boids({
  apiKey: "test-key",
  baseURL: "http://mock.test/v1",
  fetch: async (url, init = {}) => {
    const body = JSON.parse(init.body);
    calls.push({
      url,
      authorization: init.headers.Authorization,
      body,
    });

    assert.equal(url, "http://mock.test/v1/chat/complete");
    assert.equal(init.headers.Authorization, "Bearer test-key");

    if (body.stream) {
      return new Response(
        'event: chat.delta\ndata: {"delta": "Hello"}\n\n' +
          'event: chat.completed\ndata: {"id": "chat_123", "output_text": "Hello"}\n\n' +
          "data: [DONE]\n\n",
        { status: 200, headers: { "Content-Type": "text/event-stream" } },
      );
    }

    return new Response(
      JSON.stringify({
        id: "chat_123",
        message: { role: "assistant", content: "Hello" },
      }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    );
  },
});

const messages = [{ role: "user", content: "Say hello" }];

const response = await client.chat.complete({
  model: "agent:test",
  messages,
  temperature: 0.2,
});
assert.equal(response.id, "chat_123");
assert.equal(response.message.content, "Hello");

const events = [];
for await (const event of client.chat.complete({
  model: "agent:test",
  messages,
  stream: true,
})) {
  events.push(event);
}

assert.deepEqual(
  events.map((event) => event.event),
  ["chat.delta", "chat.completed"],
);
assert.equal(events[0].data.delta, "Hello");
assert.equal(events[1].data.id, "chat_123");

assert.equal(calls.length, 2);
assert.equal(calls[0].body.model, "agent:test");
assert.deepEqual(calls[0].body.messages, messages);
assert.equal(calls[0].body.stream, undefined);
assert.equal(calls[0].body.temperature, 0.2);
assert.equal(calls[1].body.stream, true);

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const cliRecords = [];
const server = createServer((request, response) => {
  let rawBody = "";
  request.setEncoding("utf8");
  request.on("data", (chunk) => {
    rawBody += chunk;
  });
  request.on("end", () => {
    const body = JSON.parse(rawBody);
    cliRecords.push({
      path: request.url,
      authorization: request.headers.authorization,
      body,
    });

    assert.equal(request.url, "/chat/complete");
    assert.equal(request.headers.authorization, "Bearer test-key");

    response.setHeader("Content-Type", "application/json");
    response.end(
      JSON.stringify({
        id: "chat_cli_123",
        message: { role: "assistant", content: "Hello from CLI" },
      }),
    );
  });
});

await new Promise((resolveListen) => {
  server.listen(0, "127.0.0.1", resolveListen);
});

try {
  const { port } = server.address();
  const cli = await runCommand(process.execPath, [
    resolve(repoRoot, "js/bin/boids.js"),
    "chat",
    "--api-key",
    "test-key",
    "--base-url",
    `http://127.0.0.1:${port}`,
    "--model",
    "agent:test",
    "--input",
    "Say hello",
    "--json",
  ]);

  assert.equal(cli.status, 0, cli.stderr);
  const cliResponse = JSON.parse(cli.stdout);
  assert.equal(cliResponse.id, "chat_cli_123");
  assert.equal(cliRecords.length, 1);
  assert.equal(cliRecords[0].body.model, "agent:test");
  assert.deepEqual(cliRecords[0].body.messages, [{ role: "user", content: "Say hello" }]);
  assert.equal(cliRecords[0].body.stream, false);
} finally {
  await new Promise((resolveClose) => {
    server.close(resolveClose);
  });
}

console.log("js chat/complete test passed");

function runCommand(command, args) {
  return new Promise((resolveCommand, reject) => {
    const child = spawn(command, args, {
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";

    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk) => {
      stdout += chunk;
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk;
    });
    child.on("error", reject);
    child.on("close", (status) => {
      resolveCommand({ status, stdout, stderr });
    });
  });
}
