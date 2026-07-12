<template>
  <div class="kpi-strip">
    <el-card shadow="never" class="kpi-card">
      <div class="kpi-label">{{ t('dashboard.kpiTotalMail') }}</div>
      <div class="kpi-value">{{ totalMail }}</div>
      <div class="kpi-hint">{{ t('dashboard.kpiWindowHint', { days: windowDays }) }}</div>
    </el-card>
    <el-card shadow="never" class="kpi-card">
      <div class="kpi-label">{{ t('dashboard.kpiThreatShare') }}</div>
      <div class="kpi-value">{{ threatPct }}%</div>
      <div class="kpi-hint">{{ t('dashboard.kpiThreatHint') }}</div>
    </el-card>
    <el-card shadow="never" class="kpi-card">
      <div class="kpi-label">{{ t('dashboard.kpiServicesRunning') }}</div>
      <div class="kpi-value">{{ runningLabel }}</div>
      <div class="kpi-hint">{{ t('dashboard.kpiServicesHint') }}</div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ServiceStatus, MailStats } from '../../api/dashboard'

const { t } = useI18n()

const props = defineProps<{
  services: ServiceStatus[]
  dailyStats: MailStats[]
}>()

const windowDays = computed(() => props.dailyStats.length || 7)

const totalMail = computed(() => {
  let s = 0
  for (const x of props.dailyStats) {
    s += x.totalCount ?? 0
  }
  return s
})

const threatPct = computed(() => {
  let threat = 0
  for (const x of props.dailyStats) {
    threat += (x.spamCount ?? 0) + (x.rejectCount ?? 0) + (x.quarantineCount ?? 0)
  }
  const tot = totalMail.value
  if (tot <= 0) {
    return '0'
  }
  return ((threat / tot) * 100).toFixed(1)
})

const runningLabel = computed(() => {
  let enabled = 0
  let running = 0
  for (const s of props.services) {
    if (s.configEnabled) {
      enabled++
      if (s.status === 'running') {
        running++
      }
    }
  }
  if (enabled === 0) {
    return '—'
  }
  return `${running}/${enabled}`
})
</script>

<style scoped>
.kpi-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}

.kpi-card {
  border-radius: 12px;
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
}

.kpi-label {
  font-size: 13px;
  color: var(--foreground-muted);
  margin-bottom: 6px;
}

.kpi-value {
  font-size: 22px;
  font-weight: 600;
  color: var(--foreground);
}

.kpi-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--foreground-muted);
}
</style>
