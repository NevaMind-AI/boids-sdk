#!/usr/bin/env node
import { Boids, DEFAULT_BASE_URL, extractText } from "../src/index.js";

const env = process.env;

async function main(argv) {
  if (looksLikeShortcut(argv)) {
    return runShortcut(argv);
  }

  const command = argv.shift();

  if (!command || command === "-h" || command === "--help") {
    usage();
    return 0;
  }

  if (command === "ask") {
    const options = parseOptions(argv);
    const model = requireModel(options.model);
    const input = options._.join(" ");
    if (!input) {
      throw new Error("Missing input text.");
    }

    const client = makeClient(options);
    const result = client.responses.create({
      model,
      input,
      stream: options.stream ?? true,
      previous_response_id: options.previousResponseId,
    });
    await printResult(result, Boolean(options.json), {
      showResponseId: Boolean(options.showResponseId),
    });
    return 0;
  }

  if (command === "search") {
    const options = parseOptions(argv, { envModel: false });
    const query = options.query ?? options._.join(" ");
    if (!query) {
      throw new Error("Missing search query.");
    }

    const client = makeClient(options);
    const result = await client.market.search({
      query,
      limit: Number(options.limit ?? 5),
    });
    printSearchResult(result, Boolean(options.json));
    return 0;
  }

  if (command === "run" || command === "auto") {
    const options = parseOptions(argv, { envModel: false });
    const input = options.input ?? options._.join(" ");
    if (!input) {
      throw new Error("Missing input text.");
    }

    const client = makeClient(options);
    const result = await runWithBestAgent(client, {
      input,
      searchQuery: options.searchQuery ?? input,
      limit: Number(options.limit ?? 1),
      stream: options.stream ?? true,
      previousResponseId: options.previousResponseId,
      showAgent: !options.quietAgent && !options.json,
    });
    await printResult(result, Boolean(options.json), {
      showResponseId: Boolean(options.showResponseId),
    });
    return 0;
  }

  if (command === "responses" && argv[0] === "create") {
    argv.shift();
    const options = parseOptions(argv);
    const model = requireModel(options.model);
    const input = options.input ?? options._.join(" ");
    if (!input) {
      throw new Error("Missing --input or input text.");
    }

    const client = makeClient(options);
    const result = client.responses.create({
      model,
      input,
      stream: Boolean(options.stream),
      previous_response_id: options.previousResponseId,
    });
    await printResult(result, Boolean(options.json), {
      showResponseId: Boolean(options.showResponseId),
    });
    return 0;
  }

  throw new Error(`Unknown command: ${command}`);
}

async function runShortcut(argv) {
  const options = parseOptions(argv, { envModel: false });
  const model = options.model ?? options._.shift();
  const input = options.input ?? options._.join(" ");

  if (!model) {
    throw new Error("Missing agent model.");
  }
  if (!input) {
    throw new Error("Missing input text.");
  }

  const client = makeClient(options);
  const result = client.responses.create({
    model,
    input,
    stream: options.stream ?? true,
    previous_response_id: options.previousResponseId,
  });
  await printResult(result, Boolean(options.json), {
    showResponseId: Boolean(options.showResponseId),
  });
  return 0;
}

function makeClient(options) {
  return new Boids({
    apiKey: options.apiKey ?? env.BOIDS_API_KEY,
    baseURL: options.baseUrl ?? DEFAULT_BASE_URL,
  });
}

async function runWithBestAgent(client, options) {
  const searchResult = await client.market.search({
    query: options.searchQuery,
    limit: options.limit,
  });
  const item = marketItems(searchResult)[0];
  if (!item) {
    throw new Error(`No agents found for: ${options.searchQuery}`);
  }

  const model = agentModel(item);
  if (!model) {
    throw new Error("Best market result did not include a usable model.");
  }

  if (options.showAgent) {
    const title = item.title ?? item.id ?? model;
    console.error(`Selected agent: ${title} (${model})`);
  }

  return client.responses.create({
    model,
    input: options.input,
    stream: options.stream,
    previous_response_id: options.previousResponseId,
  });
}

function printSearchResult(result, asJSON) {
  if (asJSON) {
    console.log(JSON.stringify(result, null, 2));
    return;
  }

  const items = marketItems(result);
  if (items.length === 0) {
    console.log("No agents found.");
    return;
  }

  items.forEach((item, index) => {
    const title = item.title ?? item.id ?? "Untitled agent";
    const model = agentModel(item) ?? "unknown";
    console.log(`${index + 1}. ${title}`);
    console.log(`   model: ${model}`);
    if (item.description) {
      console.log(`   ${item.description}`);
    }
  });
}

function marketItems(result) {
  const items = result?.data?.items;
  return Array.isArray(items) ? items.filter((item) => item && typeof item === "object") : [];
}

function agentModel(item) {
  if (typeof item.model_name === "string" && item.model_name.startsWith("agent:")) {
    return item.model_name;
  }

  const agentId = item.agent_id ?? item.id;
  if (typeof agentId === "string" && agentId) {
    return `agent:${agentId}`;
  }

  if (typeof item.model_name === "string" && item.model_name) {
    return item.model_name;
  }

  return undefined;
}

async function printResult(result, asJSON, { showResponseId = false } = {}) {
  if (isAsyncIterable(result)) {
    let wroteText = false;
    let fallbackText;
    let responseId;

    for await (const event of result) {
      if (event.event?.endsWith(".completed")) {
        fallbackText = extractText(event.data);
        responseId = extractResponseId(event.data);
      }

      if (asJSON) {
        console.log(JSON.stringify(event));
        continue;
      }

      const delta = extractDelta(event.data);
      if (delta !== undefined) {
        process.stdout.write(delta);
        wroteText = true;
        continue;
      }

    }

    if (wroteText) {
      process.stdout.write("\n");
    } else if (fallbackText) {
      console.log(fallbackText);
    }
    if (showResponseId && responseId) {
      console.error(`Response ID: ${responseId}`);
    }
    return;
  }

  const value = await result;
  if (asJSON) {
    console.log(JSON.stringify(value, null, 2));
    if (showResponseId) {
      printResponseId(value);
    }
    return;
  }

  const text = extractText(value);
  console.log(text ?? JSON.stringify(value, null, 2));
  if (showResponseId) {
    printResponseId(value);
  }
}

function parseOptions(args, { envModel = true } = {}) {
  const options = { _: [] };

  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];

    if (arg === "--stream") {
      options.stream = true;
    } else if (arg === "--no-stream") {
      options.stream = false;
    } else if (arg === "--json") {
      options.json = true;
    } else if (arg === "--show-response-id") {
      options.showResponseId = true;
    } else if (arg === "--quiet-agent") {
      options.quietAgent = true;
    } else if (arg === "--show-agent") {
      options.quietAgent = false;
    } else if (arg === "-m" || arg === "--model") {
      options.model = args[++index];
    } else if (arg === "-i" || arg === "--input") {
      options.input = args[++index];
    } else if (arg === "-q" || arg === "--query") {
      options.query = args[++index];
    } else if (arg === "--search-query") {
      options.searchQuery = args[++index];
    } else if (arg === "--limit") {
      options.limit = args[++index];
    } else if (arg === "--previous-response-id" || arg === "--prev") {
      options.previousResponseId = args[++index];
    } else if (arg === "--api-key") {
      options.apiKey = args[++index];
    } else if (arg === "--base-url") {
      options.baseUrl = args[++index];
    } else if (arg.startsWith("--model=")) {
      options.model = arg.slice("--model=".length);
    } else if (arg.startsWith("--input=")) {
      options.input = arg.slice("--input=".length);
    } else if (arg.startsWith("--query=")) {
      options.query = arg.slice("--query=".length);
    } else if (arg.startsWith("--search-query=")) {
      options.searchQuery = arg.slice("--search-query=".length);
    } else if (arg.startsWith("--limit=")) {
      options.limit = arg.slice("--limit=".length);
    } else if (arg.startsWith("--previous-response-id=")) {
      options.previousResponseId = arg.slice("--previous-response-id=".length);
    } else if (arg.startsWith("--prev=")) {
      options.previousResponseId = arg.slice("--prev=".length);
    } else if (arg.startsWith("--api-key=")) {
      options.apiKey = arg.slice("--api-key=".length);
    } else if (arg.startsWith("--base-url=")) {
      options.baseUrl = arg.slice("--base-url=".length);
    } else {
      options._.push(arg);
    }
  }

  if (envModel) {
    options.model ??= env.BOIDS_MODEL;
  }
  return options;
}

function requireModel(model) {
  if (model) {
    return model;
  }
  throw new Error("Missing --model or BOIDS_MODEL.");
}

function isAsyncIterable(value) {
  return value && typeof value[Symbol.asyncIterator] === "function";
}

function extractDelta(value) {
  if (value && typeof value === "object" && typeof value.delta === "string") {
    return value.delta;
  }
  return undefined;
}

function extractResponseId(value) {
  if (value && typeof value === "object" && typeof value.id === "string") {
    return value.id;
  }
  return undefined;
}

function printResponseId(value) {
  const responseId = extractResponseId(value);
  if (responseId) {
    console.error(`Response ID: ${responseId}`);
  }
}

function looksLikeShortcut(args) {
  const first = firstPositional(args);
  return first !== undefined && !["ask", "responses", "search", "run", "auto"].includes(first);
}

function firstPositional(args) {
  let skipNext = false;
  const optionsWithValues = new Set([
    "--api-key",
    "--base-url",
    "--model",
    "-m",
    "--input",
    "-i",
    "--query",
    "-q",
    "--search-query",
    "--limit",
    "--previous-response-id",
    "--prev",
  ]);

  for (const arg of args) {
    if (skipNext) {
      skipNext = false;
      continue;
    }

    if (optionsWithValues.has(arg)) {
      skipNext = true;
      continue;
    }

    if (
      arg.startsWith("--api-key=") ||
      arg.startsWith("--base-url=") ||
      arg.startsWith("--model=") ||
      arg.startsWith("--input=") ||
      arg.startsWith("--query=") ||
      arg.startsWith("--search-query=") ||
      arg.startsWith("--limit=") ||
      arg.startsWith("--previous-response-id=") ||
      arg.startsWith("--prev=")
    ) {
      continue;
    }

    if (arg.startsWith("-")) {
      continue;
    }

    return arg;
  }

  return undefined;
}

function usage() {
  console.log(`Usage:
  boids <agent-model> <input>
  boids search <query> [--limit 5]
  boids run <input> [--search-query <query>] [--prev <response-id>]
  boids ask --model <model> [--no-stream] <input>
  boids responses create --model <model> --input <input> [--stream] [--prev <response-id>]

Environment:
  BOIDS_API_KEY   Required API key
  BOIDS_MODEL     Optional default model`);
}

main(process.argv.slice(2)).then(
  (code) => {
    process.exitCode = code;
  },
  (error) => {
    console.error(error.message);
    process.exitCode = 1;
  },
);
