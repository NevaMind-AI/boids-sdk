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
  previous_response_id?: string;
  [key: string]: unknown;
}

export interface ResponseEvent {
  event?: string;
  data: unknown;
  raw: string;
}

export interface MarketSearchParams {
  query: string;
  limit?: number;
  [key: string]: unknown;
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
  market: {
    search(params: MarketSearchParams | string): Promise<unknown>;
  };
  createResponse(params: ResponseCreateParams): Promise<unknown> | AsyncIterable<ResponseEvent>;
  searchMarket(params: MarketSearchParams | string): Promise<unknown>;
}

export function parseSSE(response: Response): AsyncIterable<ResponseEvent>;
export function extractText(value: unknown): string | undefined;

export const DEFAULT_BASE_URL: string;
