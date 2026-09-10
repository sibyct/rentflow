import { env } from '@/shared/lib/env';
import { useAuthStore } from '@/features/auth/store/authStore';

export interface ValidationDetail {
  field: string;
  message: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  /** Structured extra context from the backend's ErrorResponse.details — currently only populated for "invalid_input". */
  readonly details?: unknown;

  constructor(status: number, code: string, message: string, details?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
  }

  /** Type-narrows `details` for the one shape the backend actually sends today (validation failures). */
  validationDetails(): ValidationDetail[] | undefined {
    if (!Array.isArray(this.details)) return undefined;
    return this.details as ValidationDetail[];
  }
}

// Matches internal/transport/http/response/json.go's success envelope.
interface SuccessEnvelope<T> {
  data?: T;
  meta?: unknown;
}

// Matches internal/transport/http/response/errors.go's ErrorResponse.
// Deliberately a different shape than SuccessEnvelope, not a shared
// envelope with optional halves — the backend only ever sends one or
// the other, and res.ok reliably tells us which.
interface ErrorEnvelope {
  error: string;
  code?: string;
  details?: unknown;
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
  /** Set on the refresh call itself to avoid an infinite refresh loop. */
  skipAuthRefresh?: boolean;
}

// Refresh calls are deduplicated: if five requests 401 at once, only one
// POST /auth/refresh goes out and the rest await its result.
let refreshInFlight: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = doRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function doRefresh(): Promise<string | null> {
  try {
    const res = await fetch(`${env.apiBaseUrl}/api/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    });
    if (!res.ok) return null;

    const envelope = (await res.json()) as SuccessEnvelope<{ access_token: string; user: unknown }>;
    const token = envelope.data?.access_token ?? null;
    if (token) {
      useAuthStore.getState().setAccessToken(token);
    }
    return token;
  } catch {
    return null;
  }
}

async function requestEnvelope<T>(path: string, options: RequestOptions = {}): Promise<SuccessEnvelope<T>> {
  const { body, skipAuthRefresh, headers, ...rest } = options;

  const doFetch = (): Promise<Response> => {
    const token = useAuthStore.getState().accessToken;
    return fetch(`${env.apiBaseUrl}${path}`, {
      ...rest,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...headers,
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  };

  let res = await doFetch();

  if (res.status === 401 && !skipAuthRefresh) {
    const newToken = await refreshAccessToken();
    if (!newToken) {
      useAuthStore.getState().clear();
      throw new ApiError(401, 'unauthorized', 'Your session has expired. Please log in again.');
    }
    res = await doFetch();
  }

  if (res.status === 204) {
    return {};
  }

  if (!res.ok) {
    const envelope = (await res.json().catch(() => ({}) as ErrorEnvelope)) as ErrorEnvelope;
    throw new ApiError(
      res.status,
      envelope.code ?? 'unknown_error',
      envelope.error || 'Something went wrong. Please try again.',
      envelope.details,
    );
  }

  return (await res.json()) as SuccessEnvelope<T>;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const envelope = await requestEnvelope<T>(path, options);
  return envelope.data as T;
}

export const apiClient = {
  get: <T>(path: string, options?: RequestOptions) => request<T>(path, { ...options, method: 'GET' }),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'POST', body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'PUT', body }),
  delete: <T>(path: string, options?: RequestOptions) => request<T>(path, { ...options, method: 'DELETE' }),
  /** Like get, but also returns envelope.meta (used for paginated list endpoints). */
  getWithMeta: async <T, M>(path: string, options?: RequestOptions): Promise<{ data: T; meta: M }> => {
    const envelope = await requestEnvelope<T>(path, { ...options, method: 'GET' });
    return { data: envelope.data as T, meta: envelope.meta as M };
  },
};
