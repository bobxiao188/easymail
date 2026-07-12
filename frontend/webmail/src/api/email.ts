import apiClient from './client'
import type {
  Email,
  EmailListItem,
  PaginationParams,
  ApiResponse,
  ComposeMessage,
  DraftDetail,
  MailStats,
} from '../types'

// Get email list
export async function getEmailList(
  params: PaginationParams
): Promise<ApiResponse<{ items: EmailListItem[]; total: number }>> {
  const response = await apiClient.get<ApiResponse<{ items: EmailListItem[]; total: number }>>('/v1/messages', { params })
  return response.data
}

// Get single email
export async function getEmail(id: number): Promise<ApiResponse<Email>> {
  const response = await apiClient.get<ApiResponse<Email>>(`/v1/messages/${id}`)
  return response.data
}

// Create email (draft)
export async function createEmail(data: Partial<Email>): Promise<ApiResponse<Email>> {
  const response = await apiClient.post<ApiResponse<Email>>('/v1/messages', data)
  return response.data
}

// Update email
export async function updateEmail(id: number, data: Partial<Email>): Promise<ApiResponse<Email>> {
  const response = await apiClient.put<ApiResponse<Email>>(`/v1/messages/${id}`, data)
  return response.data
}

// Delete email
export async function deleteEmail(id: number): Promise<ApiResponse<void>> {
  const response = await apiClient.delete<ApiResponse<void>>(`/v1/messages/${id}`)
  return response.data
}

// Move email to specified folder
export async function moveEmail(id: number, folderId: number): Promise<ApiResponse<void>> {
  const response = await apiClient.post<ApiResponse<void>>(`/v1/messages/${id}/move`, { folderId })
  return response.data
}

// Mark email as read/unread
export async function markAsRead(id: number, isRead: boolean): Promise<ApiResponse<void>> {
  const response = await apiClient.patch<ApiResponse<void>>(`/v1/messages/${id}/read`, { isRead })
  return response.data
}

// Toggle email star status
export async function toggleStar(id: number): Promise<ApiResponse<{ starred: boolean }>> {
  const response = await apiClient.patch<ApiResponse<{ starred: boolean }>>(`/v1/messages/${id}/star`)
  return response.data
}

// Search emails
export async function searchEmail(params: {
  keyword: string
  folderId?: number
  startDate?: Date
  endDate?: Date
}): Promise<ApiResponse<{ items: EmailListItem[]; total: number }>> {
  const response = await apiClient.get<ApiResponse<{ items: EmailListItem[]; total: number }>>('/v1/messages/search', { params })
  return response.data
}

// Get email statistics
export async function getMailStats(): Promise<ApiResponse<MailStats>> {
  const response = await apiClient.get<ApiResponse<MailStats>>('/v1/messages/stats')
  return response.data
}

// Batch operations
export async function batchOperation(ids: number[], action: string, folderId?: number): Promise<ApiResponse<void>> {
  const data: any = { ids, action }
  if (folderId !== undefined) {
    data.folderId = folderId
  }
  const response = await apiClient.post<ApiResponse<void>>('/v1/messages/batch', data)
  return response.data
}

// Save draft
export async function saveDraft(data: ComposeMessage): Promise<ApiResponse<{ id: number }>> {
  const response = await apiClient.post<ApiResponse<{ id: number }>>('/v1/drafts', data)
  return response.data
}

// Send email
export async function sendEmail(data: ComposeMessage): Promise<ApiResponse<{ ok: boolean }>> {
  const response = await apiClient.post<ApiResponse<{ ok: boolean }>>('/v1/messages/send', data)
  return response.data
}

// Reply to email
export async function replyToEmail(id: number, data: {
  text: string
  html?: string
}): Promise<ApiResponse<{ ok: boolean }>> {
  const response = await apiClient.post<ApiResponse<{ ok: boolean }>>(`/v1/messages/${id}/reply`, data)
  return response.data
}

// Forward email
export async function forwardEmail(id: number, data: {
  text?: string
  html?: string
  to: string
  cc?: string
}): Promise<ApiResponse<{ ok: boolean }>> {
  const response = await apiClient.post<ApiResponse<{ ok: boolean }>>(`/v1/messages/${id}/forward`, data)
  return response.data
}

// Edit draft
export async function editDraft(id: number): Promise<ApiResponse<DraftDetail>> {
  const response = await apiClient.get<ApiResponse<DraftDetail>>(`/v1/drafts/${id}`)
  return response.data
}

// Update draft
export async function updateDraft(id: number, data: Partial<ComposeMessage>): Promise<ApiResponse<{ id: number }>> {
  const response = await apiClient.patch<ApiResponse<{ id: number }>>(`/v1/drafts/${id}`, data)
  return response.data
}

// Upload attachment
export async function uploadAttachment(file: File): Promise<ApiResponse<{
  id: number
  name: string
  size: number
  contentType: string
}>> {
  const formData = new FormData()
  formData.append('file', file)
  
  const response = await apiClient.post<ApiResponse<{
    id: number
    name: string
    size: number
    contentType: string
  }>>('/v1/messages/attachment/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
  return response.data
}

// Download single attachment
export async function downloadAttachment(messageId: number, index: number): Promise<Blob> {
  const response = await apiClient.get(`/v1/messages/${messageId}/attachments/${index}`, {
    responseType: 'blob'
  })
  return response.data
}

// Download all attachments (ZIP)
export async function downloadAllAttachments(messageId: number): Promise<Blob> {
  const response = await apiClient.get(`/v1/messages/${messageId}/attachments/zip`, {
    responseType: 'blob'
  })
  return response.data
}

// Get email attachments list
export async function getAttachments(messageId: number): Promise<ApiResponse<Array<{
  index: number
  name: string
  size: number
  contentType?: string
}>>> {
  const response = await apiClient.get<Array<{
    index: number
    name: string
    size: number
    contentType?: string
  }>>(`/v1/messages/${messageId}/attachments`)
  
  return {
    code: 0,
    message: 'success',
    data: response.data || []
  }
}

// Upload image (for rich text editor embedded images)
export async function uploadImage(file: File): Promise<ApiResponse<{
  url: string
  name: string
  size: number
}>> {
  const formData = new FormData()
  formData.append('file', file)
  
  const response = await apiClient.post<ApiResponse<{
    url: string
    name: string
    size: number
  }>>('/v1/messages/image/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
  return response.data
}
