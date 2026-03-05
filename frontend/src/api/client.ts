import type { ApiError, AuthTokens, RefreshRequest } from './types';

let accessToken: string | null = null;
let refreshPromise: Promise<string> | null = null;

export function setTokens(access: string, refresh: string) {
  accessToken = access;
  localStorage.setItem('refresh_token', refresh);
}

export function clearTokens() {
  accessToken = null;
  localStorage.removeItem('refresh_token');
}

export function getRefreshToken(): string | null {
  return localStorage.getItem('refresh_token');
}

async function refreshAccessToken(): Promise<string> {
  const rt = getRefreshToken();
  if (!rt) throw new Error('No refresh token');

  const body: RefreshRequest = { refresh_token: rt };
  const res = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    clearTokens();
    throw new Error('Refresh failed');
  }

  const data = (await res.json()) as { tokens: AuthTokens };
  accessToken = data.tokens.access_token;
  localStorage.setItem('refresh_token', data.tokens.refresh_token);
  return accessToken;
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json');
  }
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  let res = await fetch(path, { ...options, headers });

  if (res.status === 401 && getRefreshToken()) {
    // Deduplicate concurrent refresh calls
    if (!refreshPromise) {
      refreshPromise = refreshAccessToken().finally(() => {
        refreshPromise = null;
      });
    }

    try {
      const newToken = await refreshPromise;
      headers.set('Authorization', `Bearer ${newToken}`);
      res = await fetch(path, { ...options, headers });
    } catch {
      clearTokens();
      window.location.href = '/login';
      throw new Error('Session expired');
    }
  }

  if (!res.ok) {
    const err: ApiError = await res.json().catch(() => ({
      code: 'UNKNOWN',
      message: res.statusText,
    }));
    throw err;
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}
