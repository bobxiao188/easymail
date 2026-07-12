<template>
  <div class="dashboard" v-loading="loading">
    <div class="dashboard-header">
      <div class="dashboard-header-main">
        <h1 class="dashboard-title">{{ t('menu.dashboard') }}</h1>
        <p class="dashboard-subtitle">{{ t('dashboard.subtitle') }}</p>
        <p v-if="lastUpdated" class="dashboard-meta">
          {{ t('dashboard.dataAsOf') }}: {{ formatGeneratedAt(lastUpdated) }}
        </p>
      </div>
      <div class="dashboard-header-actions">
        <el-button type="primary" @click="loadAll()" :loading="loading">
          <el-icon><Refresh /></el-icon>
          {{ t('dashboard.refreshAll') }}
        </el-button>
        <span class="auto-label">{{ t('dashboard.autoRefresh') }}</span>
        <el-switch v-model="autoRefreshEnabled" />
        <el-select
          v-if="autoRefreshEnabled"
          v-model="autoRefreshSec"
          size="small"
          style="width: 120px"
        >
          <el-option :label="t('dashboard.auto30s')" :value="30" />
          <el-option :label="t('dashboard.auto60s')" :value="60" />
          <el-option :label="t('dashboard.auto2m')" :value="120" />
          <el-option :label="t('dashboard.auto5m')" :value="300" />
        </el-select>
      </div>
    </div>

    <KpiStrip :services="services" :daily-stats="dailyStats" />


    <div class="dashboard-section">
      <div class="section-header">
        <h2 class="section-title">{{ t('dashboard.serviceStatus') }}</h2>
      </div>
      <ServiceStatusCard :services="services" />
    </div>

    <el-card shadow="never" class="controls-card">
      <div class="controls-row">
        <span class="control-label">{{ t('dashboard.statsRangeDays') }}</span>
        <el-select v-model="statsDays" size="small" style="width: 100px" @change="loadAll()">
          <el-option :label="'7'" :value="7" />
          <el-option :label="'14'" :value="14" />
          <el-option :label="'30'" :value="30" />
        </el-select>
        <span class="control-label">{{ t('dashboard.statsRangeMonths') }}</span>
        <el-select v-model="statsMonths" size="small" style="width: 100px" @change="loadAll()">
          <el-option :label="'6'" :value="6" />
          <el-option :label="'12'" :value="12" />
        </el-select>
        <span class="control-label">{{ t('dashboard.topHours') }}</span>
        <el-select v-model="topHours" size="small" style="width: 100px" @change="loadAll()">
          <el-option label="24" :value="24" />
          <el-option label="48" :value="48" />
          <el-option label="168" :value="168" />
        </el-select>
        <span class="control-label">{{ t('dashboard.topLimit') }}</span>
        <el-select v-model="topLimit" size="small" style="width: 100px" @change="loadAll()">
          <el-option label="10" :value="10" />
          <el-option label="20" :value="20" />
          <el-option label="50" :value="50" />
        </el-select>
      </div>
    </el-card>
    <div class="dashboard-section">
      <div class="section-header">
        <h2 class="section-title">{{ t('dashboard.mailStatsDelivery') }}</h2>
        <el-radio-group v-model="statsPeriod" class="stats-period">
          <el-radio value="daily">{{ t('dashboard.daily') }}</el-radio>
          <el-radio value="monthly">{{ t('dashboard.monthly') }}</el-radio>
        </el-radio-group>
      </div>
      <p class="section-hint">{{ t('dashboard.mailStatsDeliveryHint') }}</p>
      <MailStatsChart :stats="currentStats" :period="statsPeriod" />
    </div>

    <div class="dashboard-section">
      <div class="section-header">
        <h2 class="section-title">{{ t('dashboard.scannerPolicyStats') }}</h2>
      </div>
      <p class="section-hint">{{ t('dashboard.scannerPolicyHint') }}</p>
      <MailStatsChart :stats="scannerPolicyDaily" period="daily" :count-only="true" />
    </div>

    <div class="dashboard-section">
      <div class="section-header">
        <h2 class="section-title">{{ t('dashboard.topSenders') }}</h2>
        <div class="top-meta">{{ t('dashboard.topWindowHint', { hours: topHours, limit: topLimit }) }}</div>
        <el-radio-group v-model="topSenderType" class="top-sender-type">
          <el-radio value="domain">{{ t('dashboard.byDomain') }}</el-radio>
          <el-radio value="address">{{ t('dashboard.byAddress') }}</el-radio>
        </el-radio-group>
      </div>
      <TopSendersTable :senders="currentTopSenders" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh } from '@element-plus/icons-vue'
import { useDashboard } from '../composables/useDashboard'
import ServiceStatusCard from '../components/dashboard/ServiceStatusCard.vue'
import MailStatsChart from '../components/dashboard/MailStatsChart.vue'
import TopSendersTable from '../components/dashboard/TopSendersTable.vue'
import KpiStrip from '../components/dashboard/KpiStrip.vue'

const { t } = useI18n()

const {
  loading,
  lastUpdated,
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
} = useDashboard()

const statsPeriod = ref<'daily' | 'monthly'>('daily')
const topSenderType = ref<'domain' | 'address'>('domain')

const currentStats = computed(() => (statsPeriod.value === 'daily' ? dailyStats.value : monthlyStats.value))
const currentTopSenders = computed(() =>
  topSenderType.value === 'domain' ? topSendersByDomain.value : topSendersByAddress.value
)

function formatGeneratedAt(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) {
    return iso
  }
  return d.toLocaleString()
}

onMounted(() => {
  void loadAll()
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
  background: var(--surface);
  min-height: calc(100vh - 60px);
}

.dashboard-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
}

.dashboard-header-main {
  flex: 1;
  min-width: 200px;
}

.dashboard-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--foreground);
  margin: 0 0 8px 0;
}

.dashboard-subtitle {
  font-size: 14px;
  color: var(--foreground-muted);
  margin: 0;
}

.dashboard-meta {
  font-size: 12px;
  color: var(--foreground-muted);
  margin: 8px 0 0 0;
}

.dashboard-header-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.auto-label {
  font-size: 13px;
  color: var(--foreground-muted);
}

.controls-card {
  margin-bottom: 20px;
  border-radius: 12px;
  border: 1px solid var(--border-default);
  background: var(--surface);
}

.controls-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 16px;
}

.control-label {
  font-size: 13px;
  color: var(--foreground-muted);
}

.dashboard-section {
  background: var(--surface-elevated);
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
  border: 1px solid var(--border-default);
}

.section-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--foreground);
  margin: 0;
}

.section-hint {
  font-size: 12px;
  color: var(--foreground-muted);
  margin: 0 0 16px 0;
}

.top-meta {
  font-size: 12px;
  color: var(--foreground-muted);
  margin-right: auto;
}

.stats-period,
.top-sender-type {
  display: flex;
  gap: 10px;
}
</style>
