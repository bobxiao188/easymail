import apiClient from './client'
import type { Contact, ApiResponse, ContactGroup, ContactInput } from '../types'

// Pagination params interface
export interface ContactListParams {
  q?: string
  groupId?: string
  page?: number
  pageSize?: number
}

// Contact list response (with pagination info)
export interface ContactListResponse {
  items: Contact[]
  total: number
  page: number
  pageSize: number
}

// Get contacts list
export async function getContacts(params?: ContactListParams): Promise<ApiResponse<ContactListResponse>> {
  const response = await apiClient.get<ApiResponse<ContactListResponse>>('/v1/contacts', { params })
  return response.data
}

// Get single contact
export async function getContact(id: string): Promise<ApiResponse<Contact>> {
  const response = await apiClient.get<ApiResponse<Contact>>(`/v1/contacts/${id}`)
  return response.data
}

// Create contact
export async function createContact(data: ContactInput): Promise<ApiResponse<Contact>> {
  const response = await apiClient.post<ApiResponse<Contact>>('/v1/contacts', data)
  return response.data
}

// Update contact
export async function updateContact(id: string, data: ContactInput): Promise<ApiResponse<Contact>> {
  const response = await apiClient.patch<ApiResponse<Contact>>(`/v1/contacts/${id}`, data)
  return response.data
}

// Delete contact
export async function deleteContact(id: string): Promise<ApiResponse<void>> {
  const response = await apiClient.delete<ApiResponse<void>>(`/v1/contacts/${id}`)
  return response.data
}

// Batch delete contacts
export async function batchDeleteContacts(ids: string[]): Promise<ApiResponse<void>> {
  const response = await apiClient.delete<ApiResponse<void>>('/v1/contacts/batch', { data: { ids } })
  return response.data
}

// Import contacts from emails
export async function importContactsFromEmails(options: {
  includeSenders?: boolean
  includeRecipients?: boolean
  folder?: string
}): Promise<ApiResponse<{ importedCount: number }>> {
  const response = await apiClient.post<ApiResponse<{ importedCount: number }>>('/v1/contacts/import-from-emails', options)
  return response.data
}

// Get contact groups list
export async function getContactGroups(params?: { keyword?: string }): Promise<ApiResponse<ContactGroup[]>> {
  const response = await apiClient.get<ApiResponse<ContactGroup[]>>('/v1/contact-groups', { params })
  return response.data
}

// Create contact group
export async function createContactGroup(groupName: string): Promise<ApiResponse<ContactGroup>> {
  const response = await apiClient.post<ApiResponse<ContactGroup>>('/v1/contact-groups', { groupName })
  return response.data
}

// Update contact group
export async function updateContactGroup(id: string, groupName: string): Promise<ApiResponse<ContactGroup>> {
  const response = await apiClient.patch<ApiResponse<ContactGroup>>(`/v1/contact-groups/${id}`, { groupName })
  return response.data
}

// Delete contact group
export async function deleteContactGroup(id: string): Promise<ApiResponse<void>> {
  const response = await apiClient.delete<ApiResponse<void>>(`/v1/contact-groups/${id}`)
  return response.data
}

// Move contacts to different address book
export async function moveContactsToGroup(contactIds: string[], groupId: string): Promise<ApiResponse<void>> {
  const response = await apiClient.patch<ApiResponse<void>>('/v1/contacts/batch/move', { contactIds, groupId })
  return response.data
}
