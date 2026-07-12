import apiClient from './client'
import type { ApiResponse } from '../types'

// Folder type
export interface Folder {
  id: number
  name: string
  imapName: string
  kind: number
  unreadCount?: number
}

// Get folder list
export async function listFolders(): Promise<ApiResponse<Folder[]>> {
  const response = await apiClient.get<ApiResponse<Folder[]>>('/v1/folders')
  return response.data
}

// Create folder
export async function createFolder(name: string): Promise<ApiResponse<Folder>> {
  const response = await apiClient.post<ApiResponse<Folder>>('/v1/folders', { name })
  return response.data
}

// Rename folder
export async function renameFolder(id: number, name: string): Promise<ApiResponse<void>> {
  const response = await apiClient.patch<ApiResponse<void>>(`/v1/folders/${id}`, { name })
  return response.data
}

// Delete folder
export async function deleteFolder(id: number): Promise<ApiResponse<void>> {
  const response = await apiClient.delete<ApiResponse<void>>(`/v1/folders/${id}`)
  return response.data
}
