import { apiFetch } from './client';
import type {
  OutputTaskListResponse,
  SubmissionResponse,
  SubmitWritingRequest,
} from './types';

export async function listOutputTasks(lessonId: string): Promise<OutputTaskListResponse> {
  return apiFetch<OutputTaskListResponse>(`/api/v1/lessons/${lessonId}/output-tasks`);
}

export async function submitWriting(
  taskId: string,
  req: SubmitWritingRequest,
): Promise<SubmissionResponse> {
  return apiFetch<SubmissionResponse>(`/api/v1/output-tasks/${taskId}/submit`, {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

export async function getSubmission(submissionId: string): Promise<SubmissionResponse> {
  return apiFetch<SubmissionResponse>(`/api/v1/output-tasks/submissions/${submissionId}`);
}
