import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface GenerateEmailResponse {
  address: string;
  expires_at: string;
}

export interface EmailSummary {
  id: string;
  from: string;
  subject: string;
  preview: string;
  received_at: string;
  has_attachments: boolean;
}

export interface EmailListResponse {
  emails: EmailSummary[];
}

export interface AttachmentInfo {
  id: string;
  filename: string;
}

export interface EmailContentResponse {
  id: string;
  from: string;
  subject: string;
  body_html: string;
  body_text: string;
  received_at: string;
  attachments: AttachmentInfo[];
}

export async function generateEmail(): Promise<GenerateEmailResponse> {
  const response = await apiClient.get<GenerateEmailResponse>('/api/v1/generate');
  return response.data;
}

export async function getEmails(address: string): Promise<EmailListResponse> {
  const response = await apiClient.get<EmailListResponse>(`/api/v1/emails/${address}`);
  return response.data;
}

export async function getEmailContent(
  address: string,
  id: string
): Promise<EmailContentResponse> {
  const response = await apiClient.get<EmailContentResponse>(
    `/api/v1/email/${address}/${id}`
  );
  return response.data;
}

export async function getAttachments(
  address: string,
  id: string
): Promise<{ files: AttachmentInfo[] }> {
  const response = await apiClient.get<{ files: AttachmentInfo[] }>(
    `/api/v1/email/${address}/${id}/attachments`
  );
  return response.data;
}
