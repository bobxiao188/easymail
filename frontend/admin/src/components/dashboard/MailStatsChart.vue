<template>
  <div class="mail-stats-chart">
    <div v-if="stats.length === 0" class="empty-state">
      <el-icon class="empty-icon"><TrendCharts /></el-icon>
      <p>{{ t('dashboard.noStats') }}</p>
    </div>
    <div v-else class="stats-tabs">
      <el-tabs
        v-model="activeTab"
        class="stats-tabs-inner"
        :class="{ 'hide-tab-header': countOnly }"
      >
        <el-tab-pane :label="t('dashboard.count')" name="count">
          <div class="chart-container">
            <div class="chart-bars">
              <div
                v-for="stat in stats"
                :key="stat.date"
                class="bar-group"
              >
                <div class="bar-date">{{ formatDate(stat.date) }}</div>
                <div class="bars">
                  <div
                    class="bar bar-normal"
                    :style="{ height: getBarHeight(stat.normalCount, maxCount) + '%' }"
                    :title="t('dashboard.normalMail') + ': ' + stat.normalCount"
                  >
                    <span v-if="showBarLabel(stat.normalCount, false)" class="bar-value">{{ stat.normalCount }}</span>
                  </div>
                  <div
                    class="bar bar-spam"
                    :style="{ height: getBarHeight(stat.spamCount, maxCount) + '%' }"
                    :title="t('dashboard.spamMail') + ': ' + stat.spamCount"
                  >
                    <span v-if="showBarLabel(stat.spamCount, false)" class="bar-value">{{ stat.spamCount }}</span>
                  </div>
                  <div
                    class="bar bar-reject"
                    :style="{ height: getBarHeight(stat.rejectCount, maxCount) + '%' }"
                    :title="t('dashboard.rejectMail') + ': ' + stat.rejectCount"
                  >
                    <span v-if="showBarLabel(stat.rejectCount, false)" class="bar-value">{{ stat.rejectCount }}</span>
                  </div>
                  <div
                    class="bar bar-quarantine"
                    :style="{ height: getBarHeight(stat.quarantineCount, maxCount) + '%' }"
                    :title="t('dashboard.quarantineMail') + ': ' + stat.quarantineCount"
                  >
                    <span v-if="showBarLabel(stat.quarantineCount, false)" class="bar-value">{{ stat.quarantineCount }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="chart-legend">
            <div class="legend-item">
              <span class="legend-color normal"></span>
              <span>{{ t('dashboard.normalMail') }}</span>
            </div>
            <div class="legend-item">
              <span class="legend-color spam"></span>
              <span>{{ t('dashboard.spamMail') }}</span>
            </div>
            <div class="legend-item">
              <span class="legend-color reject"></span>
              <span>{{ t('dashboard.rejectMail') }}</span>
            </div>
            <div class="legend-item">
              <span class="legend-color quarantine"></span>
              <span>{{ t('dashboard.quarantineMail') }}</span>
            </div>
          </div>
        </el-tab-pane>
        <el-tab-pane v-if="!countOnly" :label="t('dashboard.size')" name="size">
          <div class="chart-container">
            <div class="chart-bars">
              <div
                v-for="stat in stats"
                :key="stat.date"
                class="bar-group"
              >
                <div class="bar-date">{{ formatDate(stat.date) }}</div>
                <div class="bars">
                  <div
                    class="bar bar-normal"
                    :style="{ height: getBarHeight(stat.normalSize, maxSize) + '%' }"
                    :title="t('dashboard.normalMail') + ': ' + formatSize(stat.normalSize)"
                  >
                    <span v-if="showBarLabel(stat.normalSize, true)" class="bar-value">{{ formatSizeShort(stat.normalSize) }}</span>
                  </div>
                  <div
                    class="bar bar-spam"
                    :style="{ height: getBarHeight(stat.spamSize, maxSize) + '%' }"
                    :title="t('dashboard.spamMail') + ': ' + formatSize(stat.spamSize)"
                  >
                    <span v-if="showBarLabel(stat.spamSize, true)" class="bar-value">{{ formatSizeShort(stat.spamSize) }}</span>
                  </div>
                  <div
                    class="bar bar-reject"
                    :style="{ height: getBarHeight(stat.rejectSize, maxSize) + '%' }"
                    :title="t('dashboard.rejectMail') + ': ' + formatSize(stat.rejectSize)"
                  >
                    <span v-if="showBarLabel(stat.rejectSize, true)" class="bar-value">{{ formatSizeShort(stat.rejectSize) }}</span>
                  </div>
                  <div
                    class="bar bar-quarantine"
                    :style="{ height: getBarHeight(stat.quarantineSize, maxSize) + '%' }"
                    :title="t('dashboard.quarantineMail') + ': ' + formatSize(stat.quarantineSize)"
                  >
                    <span v-if="showBarLabel(stat.quarantineSize, true)" class="bar-value">{{ formatSizeShort(stat.quarantineSize) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="chart-legend">
            <div class="legend-item">
              <span class="legend-color normal"></span>
              <span>{{ t('dashboard.normalMail') }}</span>
            </div>
            <div class="legend-item">
              <span class="legend-color spam"></span>
              <span>{{ t('dashboard.spamMail') }}</span>
            </div>
            <div class="legend-item">
              <span class="legend-color reject"></span>
              <span>{{ t('dashboard.rejectMail') }}</span>
            </div>
            <div class="legend-item">
              <span class="legend-color quarantine"></span>
              <span>{{ t('dashboard.quarantineMail') }}</span>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { TrendCharts } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { MailStats } from '../../api/dashboard'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    stats: MailStats[]
    period: 'daily' | 'monthly'
    countOnly?: boolean
  }>(),
  { countOnly: false }
)

const activeTab = ref('count')

watch(
  () => props.countOnly,
  (c) => {
    if (c) {
      activeTab.value = 'count'
    }
  },
  { immediate: true }
)

const maxCount = computed(() => {
  let max = 0
  props.stats.forEach((stat) => {
    max = Math.max(max, stat.totalCount)
  })
  return max || 1
})

const maxSize = computed(() => {
  let max = 0
  props.stats.forEach((stat) => {
    max = Math.max(max, stat.totalSize)
  })
  return max || 1
})

function getBarHeight(value: number, max: number) {
  return (value / max) * 100
}

function showBarLabel(value: number, forSize: boolean) {
  const max = forSize ? maxSize.value : maxCount.value
  return value > 0 && value >= max * 0.1
}

function formatDate(dateStr: string) {
  if (!dateStr) {
    return ''
  }
  if (/^\d{4}-\d{2}$/.test(dateStr)) {
    const [y, m] = dateStr.split('-')
    return `${y}/${m}`
  }
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) {
    return dateStr
  }
  if (props.period === 'daily') {
    return `${date.getMonth() + 1}/${date.getDate()}`
  }
  return `${date.getFullYear()}/${date.getMonth() + 1}`
}

function formatSize(bytes: number) {
  if (bytes === 0) {
    return '0 B'
  }
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function formatSizeShort(bytes: number) {
  if (bytes === 0) {
    return '0'
  }
  const k = 1024
  const sizes = ['', 'K', 'M', 'G']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))}${sizes[i]}`
}
</script>

<style scoped>
.mail-stats-chart {
  position: relative;
}

.stats-tabs {
  width: 100%;
}

.stats-tabs-inner {
  --el-tabs-header-height: 40px;
}

.hide-tab-header :deep(.el-tabs__header) {
  display: none;
}

.chart-container {
  padding: 20px 0;
}

.chart-bars {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  height: 200px;
  padding: 0 20px;
  gap: 10px;
}

.bar-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}

.bar-date {
  font-size: 11px;
  color: var(--foreground-muted);
  margin-bottom: 8px;
  writing-mode: vertical-rl;
  text-orientation: mixed;
  height: 40px;
  display: flex;
  align-items: center;
}

.bars {
  flex: 1;
  width: 100%;
  display: flex;
  gap: 3px;
  align-items: flex-end;
  justify-content: center;
}

.bar {
  flex: 1;
  min-width: 8px;
  max-width: 20px;
  border-radius: 4px 4px 0 0;
  position: relative;
  transition: height 0.3s ease;
}

.bar-normal {
  background: var(--success);
}

.bar-spam {
  background: var(--warning);
}

.bar-reject {
  background: var(--danger);
}

.bar-quarantine {
  background: var(--info);
}

.bar-value {
  position: absolute;
  top: -20px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 10px;
  color: var(--foreground-muted);
  white-space: nowrap;
}

.chart-legend {
  display: flex;
  justify-content: center;
  gap: 20px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--border-default);
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--foreground-muted);
}

.legend-color {
  width: 12px;
  height: 12px;
  border-radius: 2px;
}

.legend-color.normal {
  background: var(--success);
}

.legend-color.spam {
  background: var(--warning);
}

.legend-color.reject {
  background: var(--danger);
}

.legend-color.quarantine {
  background: var(--info);
}

.empty-state {
  text-align: center;
  padding: 60px;
}

.empty-icon {
  font-size: 48px;
  color: var(--foreground-muted);
  margin-bottom: 16px;
}

.empty-state p {
  color: var(--foreground-muted);
  margin: 0;
}
</style>