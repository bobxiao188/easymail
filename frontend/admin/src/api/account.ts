import api from './http'
import type { PageResponse } from './index'

export interface Account {
  id: number
  username: string
  domainId: number
  domain?: {
    id: number
    name: string
    description: string
    active: boolean
  }
  isDeleted: boolean
  storageQuota: number
  active: boolean
  passwordExpireTime: string | null
  createTime: string
  updateTime: string
}

export const accountApi = {
  list: (params?: { keyword?: string; page?: number; pageSize?: number; domainId?: string; status?: number }) => {
    return api.get('/v1/admin/accounts', { params }) as Promise<PageResponse<Account>>
  },

  get: (id: number) => {
    return api.get(`/v1/admin/accounts/${id}`) as Promise<{ code: number; message: string; data: Account }>
  },

  create: (data: { username: string; domainId: string; password: string; storageQuota?: number; passwordExpireTime?: string | Date | null }) => {
    return api.post('/v1/admin/accounts', data) as Promise<{ code: number; message: string }>
  },

  update: (id: number, data: { username?: string; domainId?: string; active?: boolean; storageQuota?: number; passwordExpireTime?: string | Date | null }) => {
    return api.put(`/v1/admin/accounts/${id}`, data) as Promise<{ code: number; message: string }>
  },

  delete: (id: number) => {
    return api.delete(`/v1/admin/accounts/${id}`) as Promise<{ code: number; message: string }>
  },

  setPassword: (id: number, password: string) => {
    return api.put(`/v1/admin/accounts/${id}/password`, { password }) as Promise<{ code: number; message: string }>
  },

  purge: (id: number) => {
    return api.delete(`/v1/admin/accounts/${id}/purge`) as Promise<{ code: number; message: string }>
  }
}

export default accountApi