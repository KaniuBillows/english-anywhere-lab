import { apiFetch, setTokens } from './client';
import type { AuthResponse, LoginRequest, RegisterRequest } from './types';

export async function register(req: RegisterRequest): Promise<AuthResponse> {
  const data = await apiFetch<AuthResponse>('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify(req),
  });
  setTokens(data.tokens.access_token, data.tokens.refresh_token);
  return data;
}

export async function login(req: LoginRequest): Promise<AuthResponse> {
  const data = await apiFetch<AuthResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify(req),
  });
  setTokens(data.tokens.access_token, data.tokens.refresh_token);
  return data;
}
