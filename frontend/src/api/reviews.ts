import { apiFetch } from './client';
import type {
  ReviewQueueResponse,
  ReviewSubmitRequest,
  ReviewSubmitResponse,
} from './types';

export async function getReviewQueue(limit = 20): Promise<ReviewQueueResponse> {
  return apiFetch<ReviewQueueResponse>(
    `/api/v1/reviews/queue?limit=${limit}`,
  );
}

export async function submitReview(
  req: ReviewSubmitRequest,
  idempotencyKey: string,
): Promise<ReviewSubmitResponse> {
  return apiFetch<ReviewSubmitResponse>('/api/v1/reviews/submit', {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
    body: JSON.stringify(req),
  });
}
