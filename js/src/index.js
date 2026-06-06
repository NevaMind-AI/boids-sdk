const DEFAULT_BASE_URL = "https://api.boids.so/v1";
const USER_AGENT = "boids-sdk-js/0.1.2";

export class BoidsError extends Error {
  constructor(message) {
    super(message);
    this.name = "BoidsError";
  }
}

export class BoidsAPIError extends BoidsError {
  constructor(status, body) {
    super(`Boids API error ${status}: ${body}`);
    this.name = "BoidsAPIError";
    this.status = status;
    this.body = body;
  }
}

export class Boids {
  constructor(options = {}) {
    const env = typeof process === "undefined" ? {} : process.env;

    this.apiKey = options.apiKey ?? env.BOIDS_API_KEY;
    this.baseURL = (options.baseURL ?? DEFAULT_BASE_URL).replace(/\/+$/, "");
    this.fetch = options.fetch ?? globalThis.fetch;
    this.headers = options.headers ?? {};

    this.responses = {
      create: (params) => this.createResponse(params),
    };
    this.chat = {
      complete: (params) => this.completeChat(params),
    };
    this.market = {
      search: (params) => this.searchMarket(params),
    };
  }

  createResponse(params = {}) {
    const body = withoutUndefined({ ...params });
    if (body.stream) {
      return this.streamResponse(body);
    }
    return this.requestJSON("/responses", body);
  }

  completeChat(params = {}) {
    const body = withoutUndefined({ ...params });
    if (body.stream) {
      return this.streamChatCompletion(body);
    }
    return this.requestJSON("/chat/complete", body);
  }

  async requestJSON(path, body) {
    const response = await this.request(path, body);
    const text = await response.text();
    if (!text) {
      return null;
    }

    try {
      return JSON.parse(text);
    } catch {
      return text;
    }
  }

  searchMarket(params = {}) {
    const body = typeof params === "string" ? { query: params } : { ...params };
    body.limit ??= 5;
    return this.requestJSON("/market/search", body);
  }

  async *streamResponse(body) {
    const response = await this.request("/responses", { ...body, stream: true });
    yield* parseSSE(response);
  }

  async *streamChatCompletion(body) {
    const response = await this.request("/chat/complete", { ...body, stream: true });
    yield* parseSSE(response);
  }

  async request(path, body) {
    if (!this.apiKey) {
      throw new BoidsError("Missing BOIDS_API_KEY or apiKey.");
    }
    if (!this.fetch) {
      throw new BoidsError("No fetch implementation available.");
    }

    const response = await this.fetch(`${this.baseURL}${path}`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
        "Content-Type": "application/json",
        Accept: "text/event-stream, application/json",
        "User-Agent": USER_AGENT,
        ...this.headers,
      },
      body: JSON.stringify(withoutUndefined(body)),
    });

    if (!response.ok) {
      throw new BoidsAPIError(response.status, await response.text());
    }

    return response;
  }
}

export async function* parseSSE(response) {
  const decoder = new TextDecoder();
  let buffer = "";

  for await (const chunk of readBody(response.body)) {
    buffer += decoder.decode(chunk, { stream: true });
    const parts = buffer.split(/\r?\n\r?\n/);
    buffer = parts.pop() ?? "";

    for (const block of parts) {
      const event = parseSSEBlock(block);
      if (!event) {
        continue;
      }
      if (event.raw === "[DONE]") {
        return;
      }
      yield event;
    }
  }

  buffer += decoder.decode();
  if (buffer.trim()) {
    const event = parseSSEBlock(buffer);
    if (event && event.raw !== "[DONE]") {
      yield event;
    }
  }
}

async function* readBody(body) {
  if (!body) {
    return;
  }

  if (typeof body.getReader === "function") {
    const reader = body.getReader();
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          break;
        }
        if (value) {
          yield value;
        }
      }
    } finally {
      reader.releaseLock();
    }
    return;
  }

  for await (const chunk of body) {
    yield chunk;
  }
}

function parseSSEBlock(block) {
  const lines = block.split(/\r?\n/);
  const dataLines = [];
  let eventName;

  for (const line of lines) {
    if (!line || line.startsWith(":")) {
      continue;
    }

    const index = line.indexOf(":");
    const field = index === -1 ? line : line.slice(0, index);
    let value = index === -1 ? "" : line.slice(index + 1);
    if (value.startsWith(" ")) {
      value = value.slice(1);
    }

    if (field === "event") {
      eventName = value;
    } else if (field === "data") {
      dataLines.push(value);
    }
  }

  if (dataLines.length === 0) {
    return undefined;
  }

  const raw = dataLines.join("\n");
  let data = raw;
  if (raw !== "[DONE]") {
    try {
      data = JSON.parse(raw);
    } catch {
      data = raw;
    }
  }

  return { event: eventName, data, raw };
}

export function extractText(value) {
  if (typeof value === "string") {
    return value;
  }

  if (Array.isArray(value)) {
    const parts = value.map(extractText).filter(Boolean);
    return parts.length > 0 ? parts.join("") : undefined;
  }

  if (value && typeof value === "object") {
    for (const key of ["delta", "text", "output_text"]) {
      if (typeof value[key] === "string") {
        return value[key];
      }
    }

    for (const key of ["content", "output", "message"]) {
      const text = extractText(value[key]);
      if (text) {
        return text;
      }
    }
  }

  return undefined;
}

function withoutUndefined(value) {
  return Object.fromEntries(
    Object.entries(value).filter(([, item]) => item !== undefined),
  );
}

export { DEFAULT_BASE_URL };
