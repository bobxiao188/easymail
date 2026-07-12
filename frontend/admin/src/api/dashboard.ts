import api from './http'

export interface ServiceStatus {
  name: string
  displayName: string
  status: 'running' | 'stopped' | 'warning' | 'unknown'
  lastHeartbeat: string
  configEnabled: boolean
  description: string
}

export interface MailStats {
  date: string
  totalCount: number
  normalCount: number
  spamCount: number
  rejectCount: number
  quarantineCount: number
  totalSize: number
  normalSize: number
  spamSize: number
  rejectSize: number
  quarantineSize: number
}

export interface TopSender {
  rank: number
  sender: string
  count: number
  type: 'domain' | 'address'
}

export interface DashboardData {
  services: ServiceStatus[]
  dailyStats: MailStats[]
  monthlyStats: MailStats[]
  topSendersByDomain: TopSender[]
  topSendersByAddress: TopSender[]
  scannerPolicyDaily?: MailStats[]
}

export interface StatsSummary extends DashboardData {
  generatedAt: string
  scannerPolicyDaily: MailStats[]
}

export interface StatsSummaryParams {
  days?: number
  months?: number
  topHours?: number
  topLimit?: number
}

export const dashboardApi = {
  getServiceStatus: () => {
    return api.get('/v1/admin/dashboard/services') as Promise<{ code: number; message: string; data: ServiceStatus[] }>
  },

  getMailStatsDaily: (days: number = 7) => {
    return api.get('/v1/admin/dashboard/mail-stats/daily', { params: { days } }) as Promise<{ code: number; message: string; data: MailStats[] }>
  },

  getMailStatsMonthly: (months: number = 6) => {
    return api.get('/v1/admin/dashboard/mail-stats/monthly', { params: { months } }) as Promise<{ code: number; message: string; data: MailStats[] }>
  },

  getTopSenders: (type: 'domain' | 'address', hours: number = 24, limit: number = 10) => {
    return api.get('/v1/admin/dashboard/top-senders', { params: { type, hours, limit } }) as Promise<{ code: number; message: string; data: TopSender[] }>
  },

  getAllData: () => {
    return api.get('/v1/admin/dashboard') as Promise<{ code: number; message: string; data: DashboardData }>
  },

  getStatsSummary: (params?: StatsSummaryParams) => {
    return api.get('/v1/admin/stats/summary', { params }) as Promise<{ code: number; message: string; data: StatsSummary }>
  }
}