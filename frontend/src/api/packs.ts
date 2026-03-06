import { apiFetch } from './client';
import type {
  GeneratePackRequest,
  GenerationJobResponse,
  GenericMessage,
  PackDetailResponse,
  PackListResponse,
} from './types';

export async function listPacks(
  params?: Record<string, string | number | undefined>,
): Promise<PackListResponse> {
  const qs = new URLSearchParams();
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== '') qs.set(k, String(v));
    }
  }
  const query = qs.toString();
  return apiFetch<PackListResponse>(`/api/v1/packs${query ? `?${query}` : ''}`);
}

export async function getPackDetail(packId: string): Promise<PackDetailResponse> {
  return apiFetch<PackDetailResponse>(`/api/v1/packs/${packId}`);
}

export async function enrollPack(packId: string): Promise<GenericMessage> {
  return apiFetch<GenericMessage>(`/api/v1/packs/${packId}/enroll`, {
    method: 'POST',
  });
}

export async function generatePack(req: GeneratePackRequest): Promise<GenerationJobResponse> {
  return apiFetch<GenerationJobResponse>('/api/v1/packs/generate', {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

export async function getGenerationJob(jobId: string): Promise<GenerationJobResponse> {
  return apiFetch<GenerationJobResponse>(`/api/v1/packs/generation-jobs/${jobId}`);
}
