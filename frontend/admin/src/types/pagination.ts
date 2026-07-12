export interface PaginationParams {
  page?: number
  pageSize?: number
  keyword?: string
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
  [key: string]: any
}

export interface PaginationMeta {
  page: number
  pageSize: number
  total: number
  totalPages: number
  hasPrev: boolean
  hasNext: boolean
}

export interface PageResponse<T = any> {
  code: number
  message: string
  data: T[]
  meta: PaginationMeta
  request_id?: string
  timestamp: number
}

export interface PageRequest {
  page?: number
  pageSize?: number
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
  keyword?: string
}

export interface ApiResponse<T = any> {
  code: number
  message: string
  data?: T
  request_id?: string
  timestamp?: number
}