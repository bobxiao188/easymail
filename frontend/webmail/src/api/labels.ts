import apiClient from './client'
import type { Label, ApiResponse } from '../types'

export async function getLabels(): Promise<ApiResponse<Label[]>> {
  const response = await apiClient.get<ApiResponse<Label[]>>('/v1/labels')
  return response.data
}

export async function createLabel(data: { name: string; color: string }): Promise<ApiResponse<Label>> {
  const response = await apiClient.post<ApiResponse<Label>>('/v1/labels', data)
  return response.data
}

export async function updateLabel(id: number, data: { name: string; color: string }): Promise<ApiResponse<void>> {
  const response = await apiClient.put<ApiResponse<void>>(`/v1/labels/${id}`, data)
  return response.data
}

export async function deleteLabel(id: number): Promise<ApiResponse<void>> {
  const response = await apiClient.delete<ApiResponse<void>>(`/v1/labels/${id}`)
  return response.data
}

export async function setEmailLabels(emailId: number, labelIds: number[]): Promise<ApiResponse<Label[]>> {
  const response = await apiClient.post<ApiResponse<Label[]>>(`/v1/messages/${emailId}/labels`, { labelIds })
  return response.data
}

export async function getEmailLabels(emailId: number): Promise<ApiResponse<Label[]>> {
  const response = await apiClient.get<ApiResponse<Label[]>>(`/v1/messages/${emailId}/labels`)
  return response.data
}
