import api from './http'
import type { PageResponse } from './index'

export interface FilterFeature {
  id: number
  featureKey: string
  label: string
  valueType: string
  description: string
  unit: string
  createdAt: string
  updatedAt: string
}

export interface FilterRule {
  id: number
  name: string
  enabled: boolean
  priority: number
  action: string
  conditionJson: string
  createdAt: string
  updatedAt: string
}

export interface ScannerScannerLog {
  id: number
  traceId: string
  ip: string
  /** Pipeline ordinal 0..6; may be absent on legacy API responses. */
  stage?: number
  queueId: string
  sender: string
  recipient: string
  subject: string
  ruleId?: number
  actionApplied: string
  featureSnapshotJson: string
  conditionTraceJson: string
  durationMs: number
  createdAt: string
}

export interface ScannerCustomFeatureDef {
  id: number
  featureKey: string
  label: string
  type: 'meta_regex' | 'composite'
  valueType: 'bool' | 'number'
  enabled: boolean
  specJson: string
  description: string
  unit: string
  createdAt: string
  updatedAt: string
}

export type FilterRuleAction = 'accept' | 'spam' | 'quarantine' | 'reject'

export const scannerApi = {
  listFeatures: (params?: { page?: number; pageSize?: number }) => {
    if (params && (params.page || params.pageSize)) {

      return api.get('/v1/admin/filter/features', { params }) as Promise<PageResponse<FilterFeature>>
    }

    return api.get('/v1/admin/filter/features') as Promise<{ code: number; message: string; data: FilterFeature[] }>
  },

  listRules: (params?: { page?: number; pageSize?: number }) => {
    if (params && (params.page || params.pageSize)) {
      return api.get('/v1/admin/filter/rules', { params }) as Promise<PageResponse<FilterRule>>
    }
    return api.get('/v1/admin/filter/rules') as Promise<{ code: number; message: string; data: FilterRule[] }>
  },

  listCustomFeatures: (params?: { page?: number; pageSize?: number }) => {
    if (params && (params.page || params.pageSize)) {
      return api.get('/v1/admin/filter/custom-features', { params }) as Promise<PageResponse<ScannerCustomFeatureDef>>
    }
    return api.get('/v1/admin/filter/custom-features') as Promise<{ code: number; message: string; data: ScannerCustomFeatureDef[] }>
  },

  getCustomFeature: (id: number) => {
    return api.get(`/v1/admin/filter/custom-features/${id}`) as Promise<{ code: number; message: string; data: ScannerCustomFeatureDef }>
  },

  createCustomFeature: (data: {
    featureKey: string
    label: string
    type: 'meta_regex' | 'composite'
    valueType: 'bool' | 'number'
    enabled: boolean
    specJson: string
    description?: string
    unit?: string
  }) => {
    return api.post('/v1/admin/filter/custom-features', data) as Promise<{ code: number; message: string; data: ScannerCustomFeatureDef }>
  },

  updateCustomFeature: (
    id: number,
    data: {
      featureKey: string
      label: string
      type: 'meta_regex' | 'composite'
      valueType: 'bool' | 'number'
      enabled: boolean
      specJson: string
      description?: string
      unit?: string
    }
  ) => {
    return api.put(`/v1/admin/filter/custom-features/${id}`, data) as Promise<{ code: number; message: string; data: ScannerCustomFeatureDef }>
  },

  patchCustomFeature: (id: number, data: Partial<{ enabled: boolean }>) => {
    return api.patch(`/v1/admin/filter/custom-features/${id}`, data) as Promise<{ code: number; message: string; data: ScannerCustomFeatureDef }>
  },

  deleteCustomFeature: (id: number) => {
    return api.delete(`/v1/admin/filter/custom-features/${id}`) as Promise<{ code: number; message: string }>
  },

  getRule: (id: number) => {
    return api.get(`/v1/admin/filter/rules/${id}`) as Promise<{ code: number; message: string; data: FilterRule }>
  },

  createRule: (data: {
    name: string
    enabled: boolean
    priority: number
    action: FilterRuleAction
    conditionJson: string
  }) => {
    return api.post('/v1/admin/filter/rules', data) as Promise<{ code: number; message: string; data: FilterRule }>
  },

  updateRule: (
    id: number,
    data: {
      name: string
      enabled: boolean
      priority: number
      action: FilterRuleAction
      conditionJson: string
    }
  ) => {
    return api.put(`/v1/admin/filter/rules/${id}`, data) as Promise<{ code: number; message: string; data: FilterRule }>
  },

  patchRule: (id: number, data: Partial<{ name: string; enabled: boolean; priority: number; action: FilterRuleAction; conditionJson: string }>) => {
    return api.patch(`/v1/admin/filter/rules/${id}`, data) as Promise<{ code: number; message: string; data: FilterRule }>
  },

  deleteRule: (id: number) => {
    return api.delete(`/v1/admin/filter/rules/${id}`) as Promise<{ code: number; message: string }>
  },

  listScannerLogs: (params: {
    page?: number
    pageSize?: number
    ip?: string
    sender?: string
    rcpt?: string
    created_from?: string
    created_to?: string
  }) => {
    return api.get('/v1/admin/filter/delivery-logs', { params }) as Promise<PageResponse<ScannerScannerLog>>
  },

  getScannerLog: (id: number) => {
    return api.get(`/v1/admin/filter/delivery-logs/${id}`) as Promise<{ code: number; message: string; data: ScannerScannerLog }>
  }
}

export default scannerApi
