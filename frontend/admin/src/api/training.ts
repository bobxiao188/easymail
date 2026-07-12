import api from './http'
import type { PageResponse } from './index'


// ========== Samples (public resources, not per-model) ==========

export interface Sample {
  id: number
  categoryId: number
  category: string
  tag: string
  text: string
  createdAt: string
  updatedAt: string
}

export interface SampleCategory {
  id: number
  name: string
  description: string
  sampleCount: number
  createdAt: string
  updatedAt: string
}

// ========== Training ==========

export interface TrainingParams {
  learningRate: number
  epoch: number
  wordNgrams: number
  dim: number
  loss: string
}

export interface TagMapping {
  sourceTag: string
  targetLabel: string
}

// A source group picks one category and one or more of its tags, with an
// optional per-group sample limit (random / first / last / middle).
export interface SourceGroup {
  category: string
  tags: string[]
  limitType: 'unlimited' | 'random' | 'first' | 'last' | 'middle'
  limitN: number
}

// A target class is a training label; its sources are public-sample groups
// whose texts become training examples for that class.
export interface CategoryMapping {
  targetClass: string
  sources: SourceGroup[]
}

export interface TrainingRequest {
  modelName: string
  algorithm: string
  params: TrainingParams
  sampleMappings: CategoryMapping[]
}

export interface TrainingTask {
  id: number
  modelName: string
  algorithm: string
  params: TrainingParams
  sampleMappings: CategoryMapping[]
  status: string
  trainResult: string
  modelId: number
  createdAt: string
  updatedAt: string
}

// ========== Training APIs ==========
export const trainingApi = {
  // ===== Samples =====
  listSamples: (params?: {
    categoryId?: number
    tag?: string
    keyword?: string
    page?: number
    pageSize?: number
  }) => {
    return api.get('/v1/admin/samples', { params }) as Promise<PageResponse<Sample>>
  },

  createSample: (data: { categoryId: number; tag: string; text: string }) => {
    return api.post('/v1/admin/samples', data) as Promise<{ code: number; message: string }>
  },

  createSamplesBatch: (data: { items: { categoryId: number; tag: string; text: string }[] }) => {
    return api.post('/v1/admin/samples/batch', data) as Promise<{ code: number; message: string }>
  },

  updateSample: (id: number, data: { categoryId?: number; tag?: string; text?: string }) => {
    return api.put(`/v1/admin/samples/${id}`, data) as Promise<{ code: number; message: string }>
  },

  deleteSample: (id: number) => {
    return api.delete(`/v1/admin/samples/${id}`) as Promise<{ code: number; message: string }>
  },

  batchDeleteSamples: (ids: number[]) => {
    return api.post('/v1/admin/samples/batch-delete', { ids }) as Promise<{ code: number; message: string }>
  },

  batchUpdateSamples: (ids: number[], categoryId: number, tag: string) => {
    return api.post('/v1/admin/samples/batch-update', { ids, categoryId, tag }) as Promise<{ code: number; message: string }>
  },

  listTags: (categoryId: number) => {
    return api.get('/v1/admin/samples/tags', { params: { categoryId } }) as Promise<{ code: number; message: string; data: string[] }>
  },

  describeSamples: (categoryId?: number) => {
    return api.get('/v1/admin/samples/stats', { params: { categoryId } }) as Promise<{ code: number; message: string; data: { categoryId: number; category: string; tag: string; count: number }[] }>
  },

  // ===== Sample Categories =====
  listSampleCategories: (params?: {
    keyword?: string
    page?: number
    pageSize?: number
  }) => {
    return api.get('/v1/admin/sample-categories', { params }) as Promise<PageResponse<SampleCategory>>
  },

  getSampleCategory: (id: number) => {
    return api.get(`/v1/admin/sample-categories/${id}`) as Promise<{ code: number; message: string; data: SampleCategory }>
  },

  createSampleCategory: (data: { name: string; description?: string }) => {
    return api.post('/v1/admin/sample-categories', data) as Promise<{ code: number; message: string; data: SampleCategory }>
  },

  updateSampleCategory: (id: number, data: { name?: string; description?: string }) => {
    return api.put(`/v1/admin/sample-categories/${id}`, data) as Promise<{ code: number; message: string; data: SampleCategory }>
  },

  deleteSampleCategory: (id: number) => {
    return api.delete(`/v1/admin/sample-categories/${id}`) as Promise<{ code: number; message: string }>
  },

  // ===== Training =====
  startTraining: (data: TrainingRequest) => {
    return api.post('/v1/admin/training', data) as Promise<{ code: number; message: string; data: TrainingTask }>
  },

  getTraining: (id: number) => {
    return api.get(`/v1/admin/training/${id}`) as Promise<{ code: number; message: string; data: TrainingTask }>
  },

  getTrainingLog: (id: number) => {
    return api.get(`/v1/admin/training/${id}/log`) as Promise<{ code: number; message: string; data: { log: string } }>
  },

}

export default trainingApi