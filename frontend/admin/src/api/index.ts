export interface ApiResponse {
  code: number
  message: string
  data?: any
  meta?: any
  request_id?: string
  timestamp?: number
}

export interface PageResponse<T = any> {
  code: number
  message: string
  data: T[]
  meta: {
    page: number
    pageSize: number
    total: number
    totalPages: number
    hasPrev: boolean
    hasNext: boolean
  }
}

export { default } from './http'
export { setUnauthorizedHandler } from './http'
