export interface BoidsOptions {
  apiKey?: string;
  baseURL?: string;
  fetch?: typeof fetch;
  headers?: Record<string, string>;
}

export interface ResponseCreateParams {
  model: string;
  input: unknown;
  stream?: boolean;
  [key: string]: unknown;
}

export interface ResponseEvent {
  event?: string;
  data: unknown;
  raw: string;
}

export class BoidsError extends Error {}

export class BoidsAPIError extends BoidsError {
  status: number;
  body: string;
}

export class Boids {
  constructor(options?: BoidsOptions);
  responses: {
    create(params: ResponseCreateParams): Promise<unknown> | AsyncIterable<ResponseEvent>;
  };
  createResponse(params: ResponseCreateParams): Promise<unknown> | AsyncIterable<ResponseEvent>;
}

export function parseSSE(response: Response): AsyncIterable<ResponseEvent>;
export function extractText(value: unknown): string | undefined;

export const DEFAULT_BASE_URL: string;
