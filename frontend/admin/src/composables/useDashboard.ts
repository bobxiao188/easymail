import { ref, watch, onUnmounted } from 'vue'
import { dashboardApi, type ServiceStatus, type MailStats, type TopSender } from '../api/dashboard'
import { messageFromApiError } from '../utils/apiError'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'

export function useDashboard() {
  const { t } = useI18n()
  const loading = ref(false)
  const lastUpdated = ref<string | null>(null)
  const loadError = ref<string | null>(null)

  const services = ref<ServiceStatus[]>([])
  const dailyStats = ref<MailStats[]>([])
  const monthlyStats = ref<MailStats[]>([])
  const scannerPolicyDaily = ref<MailStats[]>([])
  const topSendersByDomain = ref<TopSender[]>([])
  const topSendersByAddress = ref<TopSender[]>([])

  const statsDays = ref(7)
  const statsMonths = ref(6)
  const topHours = ref(24)
  const topLimit = ref(10)

  const autoRefreshEnabled = ref(false)
  const autoRefreshSec = ref(120)
  let autoTimer: ReturnType<typeof setInterval> | null = null

  function clearAutoTimer() {
    if (autoTimer != null) {
      clearInterval(autoTimer)
      autoTimer = null
    }
  }

  function applyAutoRefresh() {
    clearAutoTimer()
    if (!autoRefreshEnabled.value || autoRefreshSec.value <= 0) {
      return
    }
    autoTimer = setInterval(() => {
      void loadAll({ silent: true })
    }, autoRefreshSec.value * 1000)
  }

  watch([autoRefreshEnabled, autoRefreshSec], applyAutoRefresh)

  async function loadAll(opts?: { silent?: boolean }) {
    const silent = opts?.silent === true
    if (!silent) {
      loading.value = true
    }
    loadError.value = null
    try {
      const res = await dashboardApi.getStatsSummary({
        days: statsDays.value,
        months: statsMonths.value,
        topHours: topHours.value,
        topLimit: topLimit.value
      })
      if (res.code !== 0) {
        const msg = messageFromApiError(res, t('common.loadFailed'))
        loadError.value = msg
        if (!silent) {
          ElMessage.error(msg)
        }
        return
      }
      const d = res.data
      services.value = d.services ?? []
      dailyStats.value = d.dailyStats ?? []
      monthlyStats.value = d.monthlyStats ?? []
      scannerPolicyDaily.value = d.scannerPolicyDaily ?? []
      topSendersByDomain.value = d.topSendersByDomain ?? []
      topSendersByAddress.value = d.topSendersByAddress ?? []
      lastUpdated.value = d.generatedAt ?? null
    } catch (e) {
      const msg = messageFromApiError(e, t('common.loadFailed'))
      loadError.value = msg
      if (!silent) {
        ElMessage.error(msg)
      }
    } finally {
      if (!silent) {
        loading.value = false
      }
    }
  }

  onUnmounted(() => {
    clearAutoTimer()
  })

  return {
    loading,
    lastUpdated,
    loadError,
    services,
    dailyStats,
    monthlyStats,
    scannerPolicyDaily,
    topSendersByDomain,
    topSendersByAddress,
    statsDays,
    statsMonths,
    topHours,
    topLimit,
    autoRefreshEnabled,
    autoRefreshSec,
    loadAll
  }
}
