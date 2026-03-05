import { apiFetch } from './client';
import type { MeResponse, UpdateProfileRequest } from './types';

export async function getMe(): Promise<MeResponse> {
  return apiFetch<MeResponse>('/api/v1/me');
}

export async function updateProfile(req: UpdateProfileRequest): Promise<MeResponse> {
  return apiFetch<MeResponse>('/api/v1/me/profile', {
    method: 'PATCH',
    body: JSON.stringify(req),
  });
}
