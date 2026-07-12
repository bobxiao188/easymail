import api from './http'
import type { PageResponse } from './index'

export interface PostfixAgent {
  id: string
  name: string
  host: string
  token?: string
  enabled: boolean
  lastStatus: string
  lastSyncAt: string
  description: string
  createTime: string
  updateTime: string
}

export interface PostfixConfig {
  id: string
  paramName: string
  paramValue: string
  category: string
  isManaged: boolean
  enabled: boolean
  description: string
  sortOrder: number
  createTime: string
  updateTime: string
}

export interface ConfigPreview {
  mainCf: string
  configHash: string
  domainCount: number
}

export interface AgentStatusInfo {
  postfixRunning: boolean
  configHash: string
  lastReloadAt: string
  postfixVersion: string
  agentVersion: string
  uptime: string
}

export interface ConfigStatusSummary {
  agents: AgentConfigStatus[]
}

export interface PostfixSettings {
  easymailHost: string
}

export interface AgentConfigStatus {
  agentId: string
  agentName: string
  host: string
  online: boolean
  lastSyncAt: string
  configHash: string
  upToDate: boolean
  lastError: string
}

export interface DeliveryLog {
  id: string
  agentId: string
  action: string
  status: string
  configSnapshot: string
  errorMessage: string
  createdAt: string
  agentName: string
}

export interface QueueMessage {
  queueId: string
  size: number
  age: string
  sender: string
  recipients: string[]
  status: string
  statusText: string
}

export interface QueueStats {
  total: number
  active: number
  deferred: number
  held: number
}

export interface QueueListResponse {
  messages: QueueMessage[]
  total: number
  page: number
  pageSize: number
}

export interface QueueFilter {
  status?: string
  sender?: string
  recipient?: string
  queueId?: string
  page?: number
  pageSize?: number
}

export const postfixApi = {
  // Agents
  listAgents: (params?: { keyword?: string; page?: number; pageSize?: number }) => {
    return api.get('/v1/admin/postfix/agents', { params }) as Promise<PageResponse<PostfixAgent>>
  },
  getAgent: (id: string) => {
    return api.get(`/v1/admin/postfix/agents/${id}`) as Promise<{ code: number; message: string; data: PostfixAgent }>
  },
  createAgent: (data: { name: string; host: string; token: string; description?: string }) => {
    return api.post('/v1/admin/postfix/agents', data) as Promise<{ code: number; message: string; data: PostfixAgent }>
  },
  updateAgent: (id: string, data: { name: string; host: string; token?: string; description?: string; enabled: boolean }) => {
    return api.put(`/v1/admin/postfix/agents/${id}`, data) as Promise<{ code: number; message: string }>
  },
  deleteAgent: (id: string) => {
    return api.delete(`/v1/admin/postfix/agents/${id}`) as Promise<{ code: number; message: string }>
  },
  checkAgentStatus: (id: string) => {
    return api.get(`/v1/admin/postfix/agents/${id}/status`) as Promise<{ code: number; message: string; data: AgentStatusInfo }>
  },

  // Config params
  listConfigs: (params?: { keyword?: string; page?: number; pageSize?: number }) => {
    return api.get('/v1/admin/postfix/configs', { params }) as Promise<PageResponse<PostfixConfig>>
  },
  getConfig: (id: string) => {
    return api.get(`/v1/admin/postfix/config/${id}`) as Promise<{ code: number; message: string; data: PostfixConfig }>
  },
  createConfig: (data: { paramName: string; paramValue: string; description?: string }) => {
    return api.post('/v1/admin/postfix/configs', data) as Promise<{ code: number; message: string; data: PostfixConfig }>
  },
  updateConfig: (id: string, data: { paramValue: string }) => {
    return api.put(`/v1/admin/postfix/config/${id}`, data) as Promise<{ code: number; message: string }>
  },
  deleteConfig: (id: string) => {
    return api.delete(`/v1/admin/postfix/config/${id}`) as Promise<{ code: number; message: string }>
  },

  // Generation & delivery
  preview: () => {
    return api.get('/v1/admin/postfix/preview') as Promise<{ code: number; message: string; data: ConfigPreview }>
  },
  pushToAgent: (agentId: string) => {
    return api.post(`/v1/admin/postfix/agents/${agentId}/push`) as Promise<{ code: number; message: string }>
  },
  applyOnAgent: (agentId: string) => {
    return api.post(`/v1/admin/postfix/agents/${agentId}/apply`) as Promise<{ code: number; message: string }>
  },
  rollbackOnAgent: (agentId: string) => {
    return api.post(`/v1/admin/postfix/agents/${agentId}/rollback`) as Promise<{ code: number; message: string }>
  },
  pushAndApply: (agentId: string) => {
    return api.post(`/v1/admin/postfix/agents/${agentId}/push-and-apply`) as Promise<{ code: number; message: string }>
  },

  // Logs
  listLogs: (agentId: string, params?: { limit?: number }) => {
    return api.get(`/v1/admin/postfix/agents/${agentId}/logs`, { params }) as Promise<{ code: number; message: string; data: DeliveryLog[] }>
  },

  // Settings
  getSettings: () => {
    return api.get('/v1/admin/postfix/settings') as Promise<{ code: number; message: string; data: PostfixSettings }>
  },
  updateSettings: (data: PostfixSettings) => {
    return api.put('/v1/admin/postfix/settings', data) as Promise<{ code: number; message: string }>
  },

  // Variables
  getVariables: () => {
    return api.get('/v1/admin/postfix/variables') as Promise<{ code: number; message: string; data: Record<string, string> }>
  },

  // Local IPs
  getLocalIPs: () => {
    return api.get('/v1/admin/postfix/local-ips') as Promise<{ code: number; message: string; data: string[] }>
  },

  // Status summary
  status: () => {
    return api.get('/v1/admin/postfix/status') as Promise<{ code: number; message: string; data: ConfigStatusSummary }>
  },

  // Queue management
  listQueue: (agentId: string, params?: QueueFilter) => {
    return api.get(`/v1/admin/postfix/queue/agents/${agentId}`, { params }) as Promise<{ code: number; message: string; data: QueueListResponse }>
  },
  getQueueStats: (agentId: string) => {
    return api.get(`/v1/admin/postfix/queue/agents/${agentId}/stats`) as Promise<{ code: number; message: string; data: QueueStats }>
  },
  deleteQueueMessages: (agentId: string, messageIds: string[]) => {
    return api.post(`/v1/admin/postfix/queue/agents/${agentId}/delete`, { messageIds }) as Promise<{ code: number; message: string }>
  },
  resendQueueMessages: (agentId: string, messageIds: string[]) => {
    return api.post(`/v1/admin/postfix/queue/agents/${agentId}/resend`, { messageIds }) as Promise<{ code: number; message: string }>
  },
  flushQueue: (agentId: string) => {
    return api.post(`/v1/admin/postfix/queue/agents/${agentId}/flush`) as Promise<{ code: number; message: string }>
  }
}

export default postfixApi
