import axios from 'axios'
import api from './http'
import type { PageResponse } from './index'

function adminApiBase(): string {
  return '/api'
}

export interface OnnxModelParams {
  algorithm: string
  configJSON?: string
  modelFile?: string
  specialTokensMap?: string
  tokenizerJSON?: string
  tokenizerConfigJSON?: string
  vocabTXT?: string
  /** FastText training */
  learningRate?: number
  epoch?: number
  wordNgrams?: number
  dim?: number
  loss?: string
}

export interface ClassifyModel {
  id: number
  name: string
  algorithm: string
  tokenizer: string
  languages: string[]
  params: OnnxModelParams
  savePath: string
  maxTextLength: number
  emailFields: string[]
  /** Distinct labels from training samples; persisted and used for stable inference feature keys. */
  classLabels?: string[]
  /** Server: model files exist and load path checks pass; required to turn activation on. */
  activationReady?: boolean
  trainStatus?: string
  trainResult?: string
  trainTime?: string
  enabled: boolean
  isDeleted: boolean
  createdAt: string
  updatedAt: string
}

export interface ClassifyPredictResult {
  modelId: string
  modelName: string
  topLabel: string
  topProbability: number
  distribution: { label: string; probability: number }[]
  predictError?: string
}

export interface ModelSample {
  id: number
  modelId: number
  text: string
  label: string
  createdAt: string
  updatedAt: string
}

export const classifyModelApi = {
  list: (params?: {
    keyword?: string
    algorithm?: string
    status?: number
    page?: number
    pageSize?: number
  }) => {
    return api.get('/v1/admin/classify-models', { params }) as Promise<PageResponse<ClassifyModel>>
  },

  get: (id: number) => {
    return api.get(`/v1/admin/classify-models/${id}`) as Promise<{ code: number; message: string; data: ClassifyModel }>
  },

  /** DistilBERT: multipart create with field onnx + form fields (same request as metadata). */
  createDistilBERT: (fd: FormData) => {
    return api.post('/v1/admin/classify-models', fd, {
      headers: { 'Content-Type': undefined } as unknown as Record<string, string>,
      timeout: 120000
    }) as Promise<{ code: number; message: string }>
  },

  /** DistilBERT: multipart update; onnx optional. */
  updateDistilBERT: (id: number, fd: FormData) => {
    return api.put(`/v1/admin/classify-models/${id}`, fd, {
      headers: { 'Content-Type': undefined } as unknown as Record<string, string>,
      timeout: 120000
    }) as Promise<{ code: number; message: string }>
  },

  create: (data: {
    name: string
    algorithm: string
    tokenizer: string
    languages: string[]
    savePath?: string
    maxTextLength: number
    emailFields: string[]
    params?: OnnxModelParams
  }) => {
    return api.post('/v1/admin/classify-models', data) as Promise<{ code: number; message: string }>
  },

  update: (id: number, data: Partial<{
    name: string
    algorithm: string
    tokenizer: string
    languages: string[]
    savePath: string
    maxTextLength: number
    emailFields: string[]
    enabled: boolean
    params?: OnnxModelParams
  }>) => {
    return api.put(`/v1/admin/classify-models/${id}`, data) as Promise<{ code: number; message: string }>
  },

  delete: (id: number) => {
    return api.delete(`/v1/admin/classify-models/${id}`) as Promise<{ code: number; message: string }>
  },

  startTrain: (id: number) => {
    return api.post(`/v1/admin/classify-models/${id}/train`) as Promise<{ code: number; message: string }>
  },

  predict: (id: number, body: { text: string; languageCodes?: string[] }) => {
    return api.post(`/v1/admin/classify-models/${id}/predict`, body) as Promise<{
      code: number
      message: string
      data: ClassifyPredictResult
    }>
  },

  listSamples: (modelId: number, params?: { keyword?: string; label?: string; page?: number; pageSize?: number }) => {
    return api.get(`/v1/admin/classify-models/${modelId}/samples`, { params }) as Promise<PageResponse<ModelSample>>
  },

  listSampleLabels: (modelId: number) => {
    return api.get(`/v1/admin/classify-models/${modelId}/samples/labels`) as Promise<{ code: number; message: string; data: string[] }>
  },

  /** Download train.txt (__label__<class>\\t<text> per line, UTF-8) for scripts/train/distiBERT/train.py */
  exportSamplesTrainTxt: async (modelId: number): Promise<void> => {
    const token = sessionStorage.getItem('token')
    const res = await axios.get(`${adminApiBase()}/v1/admin/classify-models/${modelId}/samples/export`, {
      responseType: 'blob',
      validateStatus: (s) => s < 500,
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      timeout: 120000
    })
    if (res.status !== 200) {
      let msg = 'Export failed'
      try {
        const t = await (res.data as Blob).text()
        const j = JSON.parse(t) as { message?: string }
        if (j.message) msg = j.message
      } catch {
        /* ignore */
      }
      throw new Error(msg)
    }
    const dispo = String(res.headers['content-disposition'] || '')
    let name = 'train.txt'
    const mStar = /filename\*=UTF-8''([^;\s]+)/i.exec(dispo)
    const mQ = /filename="([^"]+)"/i.exec(dispo)
    if (mStar) name = decodeURIComponent(mStar[1])
    else if (mQ) name = mQ[1]
    const url = URL.createObjectURL(res.data as Blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    a.click()
    URL.revokeObjectURL(url)
  },

  createSamples: (modelId: number, body: { text?: string; label?: string; items?: { text: string; label: string }[] }) => {
    return api.post(`/v1/admin/classify-models/${modelId}/samples`, body) as Promise<{ code: number; message: string }>
  },

  updateSample: (modelId: number, sampleId: number, body: { text: string; label: string }) => {
    return api.put(`/v1/admin/classify-models/${modelId}/samples/${sampleId}`, body) as Promise<{ code: number; message: string }>
  },

  deleteSample: (modelId: number, sampleId: number) => {
    return api.delete(`/v1/admin/classify-models/${modelId}/samples/${sampleId}`) as Promise<{ code: number; message: string }>
  },

  /** Download model as zip (model.conf + binary). */
  exportModel: async (modelId: number): Promise<void> => {
    const token = sessionStorage.getItem('token')
    const res = await axios.get(`${adminApiBase()}/v1/admin/classify-models/${modelId}/export`, {
      responseType: 'blob',
      validateStatus: (s) => s < 500,
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      timeout: 120000
    })
    if (res.status !== 200) {
      let msg = 'Export failed'
      try {
        const t = await (res.data as Blob).text()
        const j = JSON.parse(t) as { message?: string }
        if (j.message) msg = j.message
      } catch { /* ignore */ }
      throw new Error(msg)
    }
    const dispo = String(res.headers['content-disposition'] || '')
    let name = 'model.zip'
    const mStar = /filename\*=UTF-8''([^;\s]+)/i.exec(dispo)
    const mQ = /filename="([^"]+)"/i.exec(dispo)
    if (mStar) name = decodeURIComponent(mStar[1])
    else if (mQ) name = mQ[1]
    const url = URL.createObjectURL(res.data as Blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    a.click()
    URL.revokeObjectURL(url)
  },

  /** Import model from a zip file upload. */
  importModel: (fd: FormData, algorithm: string) => {
    fd.append('algorithm', algorithm)
    return api.post('/v1/admin/classify-models/import', fd, {
      headers: { 'Content-Type': undefined } as unknown as Record<string, string>,
      timeout: 120000
    }) as Promise<{ code: number; message: string }>
  }
}

export default classifyModelApi
