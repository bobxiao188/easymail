import apiClient from './client'
import type { User, UserSettings, ApiResponse } from '../types'

// Login
export async function login(email: string, password: string): Promise<ApiResponse<{ user: User; token: string }>> {
  const response = await apiClient.post<ApiResponse<{ user: User; token: string }>>('/v1/auth/login', { email, password })
  return response.data
}

// Logout
export async function logout(): Promise<ApiResponse<void>> {
  const response = await apiClient.post<ApiResponse<void>>('/v1/auth/logout')
  return response.data
}

// Get current user info
export async function getCurrentUser(): Promise<ApiResponse<User>> {
  const response = await apiClient.get<ApiResponse<User>>('/v1/profile')
  return response.data
}

// Update user info
export async function updateUser(data: Partial<User>): Promise<ApiResponse<User>> {
  const response = await apiClient.put<ApiResponse<User>>('/v1/profile', data)
  return response.data
}

// Change password
export async function changePassword(data: { oldPassword: string; newPassword: string }): Promise<ApiResponse<void>> {
  const response = await apiClient.post<ApiResponse<void>>('/v1/auth/password/change', data)
  return response.data
}

// Get profile
export async function getProfile(): Promise<ApiResponse<User>> {
  const response = await apiClient.get<ApiResponse<User>>('/v1/profile')
  return response.data
}

// Update profile
export async function updateProfile(data: Partial<User>): Promise<ApiResponse<void>> {
  const response = await apiClient.put<ApiResponse<void>>('/v1/profile', data)
  return response.data
}

// Get user settings
export async function getUserSettings(): Promise<ApiResponse<UserSettings>> {
  const response = await apiClient.get<ApiResponse<UserSettings>>('/v1/settings')
  return response.data
}

// Update user settings
export async function updateUserSettings(data: Partial<UserSettings>): Promise<ApiResponse<UserSettings>> {
  const response = await apiClient.put<ApiResponse<UserSettings>>('/v1/settings', data)
  return response.data
}
