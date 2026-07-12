import api from './http'
import type { PageResponse } from './index'

export interface Domain {
  id: string
  name: string
  description: string
  active: boolean
  isDeleted: boolean
  dkimEnabled: boolean
  dkimSelector: string
  dkimPrivateKey: string
  createTime: string
  updateTime: string
}

export const domainApi = {
  list: (params?: { keyword?: string; page?: number; pageSize?: number; include_deleted?: boolean }) => {
    return api.get('/v1/admin/domains', { params }) as Promise<PageResponse<Domain>>
  },

  get: (id: string) => {
    return api.get(`/v1/admin/domains/${id}`) as Promise<{ code: number; message: string; data: Domain }>
  },

  create: (data: { name: string; description?: string }) => {
    return api.post('/v1/admin/domains', data) as Promise<{ code: number; message: string }>
  },

  update: (id: string, data: { name: string; description?: string; active?: boolean; isDeleted?: boolean }) => {
    return api.put(`/v1/admin/domains/${id}`, data) as Promise<{ code: number; message: string }>
  },

  delete: (id: string) => {
    return api.delete(`/v1/admin/domains/${id}`) as Promise<{ code: number; message: string }>
  },

  toggle: (id: string) => {
    return api.put(`/v1/admin/domains/${id}/toggle`) as Promise<{ code: number; message: string }>
  },

  updateDKIM: (id: string, data: { enabled: boolean; selector: string; privateKey: string }) => {
    return api.put(`/v1/admin/domains/${id}/dkim`, data) as Promise<{ code: number; message: string }>
  },

  purge: (id: string) => {
    return api.delete(`/v1/admin/domains/${id}/purge`) as Promise<{ code: number; message: string }>
  }
}

export default domainApi