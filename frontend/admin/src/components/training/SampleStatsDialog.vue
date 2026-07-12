<template>
  <div class="sample-stats-dialog">
    <el-button link size="default" :disabled="!categoryId" @click="handleOpen">
      {{ t('training.describeSamples') }}
    </el-button>

    <el-dialog v-model="visible" :title="t('training.describeSamples')" width="53%"
      class="easy-dialog" align-center :close-on-click-modal="false">
      <div v-loading="loading" class="stats-body">
        <template v-if="stats.length > 0">
          <!-- header row -->
          <div class="stats-header">
            <span class="stats-header-rank">{{ t('training.describeRank') }}</span>
            <span class="stats-header-label">{{ t('training.tag') }}</span>
            <span class="stats-header-track"></span>
            <span class="stats-header-count">{{ t('training.describeCount') }}</span>
            <span class="stats-header-percent">{{ t('training.describePercent') }}</span>
          </div>
          <!-- data rows -->
          <div v-for="(stat, idx) in stats" :key="stat.tag" class="stats-row">
            <span class="stats-rank">{{ idx + 1 }}</span>
            <span class="stats-label">{{ stat.tag }}</span>
            <div class="stats-track">
              <div class="stats-fill" :style="{ width: stat.percent + '%' }"></div>
            </div>
            <span class="stats-count">{{ stat.count }}</span>
            <span class="stats-percent">{{ stat.percent.toFixed(1) }}%</span>
          </div>
        </template>
        <el-empty v-else :description="t('training.describeNoData')" />
      </div>
      <template #footer>
        <el-button class="easy-button" @click="visible = false">{{ t('common.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { trainingApi } from '@/api/training'

interface Props {
  categoryId?: number | null
}

const props = defineProps<Props>()

const { t } = useI18n()

const visible = ref(false)
const loading = ref(false)

interface StatsItem {
  tag: string
  count: number
  percent: number
}

const stats = ref<StatsItem[]>([])

async function handleOpen() {
  if (!props.categoryId) return

  loading.value = true
  visible.value = true

  try {
    const res = await trainingApi.describeSamples(props.categoryId)
    if (res.code === 0 && Array.isArray(res.data)) {
      const raw = res.data as { tag: string; count: number }[]
      const total = raw.reduce((s, r) => s + r.count, 0)
      stats.value = raw
        .map(r => ({ ...r, percent: total > 0 ? (r.count / total) * 100 : 0 }))
        .sort((a, b) => b.count - a.count)
    } else {
      stats.value = []
    }
  } catch {
    ElMessage.error(t('common.loadFailed'))
    stats.value = []
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.stats-body {
  min-height: 120px;
}

.stats-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-default);
  font-size: 13px;
  color: var(--foreground-muted);
  font-weight: 500;
}

.stats-header-rank {
  width: 100px;
  text-align: center;
  flex-shrink: 0;
}

.stats-header-label {
  width: 120px;
  flex-shrink: 0;
}

.stats-header-track {
  flex: 1;
}

.stats-header-count {
  width: 60px;
  text-align: right;
  flex-shrink: 0;
}

.stats-header-percent {
  width: 100px;
  text-align: right;
  flex-shrink: 0;
}

.stats-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-subtle);
  transition: background 0.15s;
}

.stats-row:last-child {
  border-bottom: none;
}

.stats-row:hover {
  background: var(--surface-light);
  border-radius: 4px;
}

.stats-rank {
  width: 100px;
  text-align: center;
  font-size: 14px;
  font-weight: 600;
  color: var(--foreground-muted);
  flex-shrink: 0;
}

.stats-label {
  width: 120px;
  font-size: 14px;
  color: var(--foreground);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 0;
}

.stats-track {
  flex: 1;
  height: 20px;
  background: var(--surface-light);
  border-radius: 4px;
  overflow: hidden;
  min-width: 80px;
}

.stats-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent), var(--accent-light, var(--accent)));
  border-radius: 4px;
  transition: width 0.4s ease;
  min-width: 2px;
}

.stats-count {
  width: 60px;
  text-align: right;
  font-size: 14px;
  color: var(--foreground);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}

.stats-percent {
  width: 100px;
  text-align: right;
  font-size: 13px;
  color: var(--foreground-muted);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
</style>

<style>
.sample-stats-dialog .easy-dialog {
  --el-dialog-bg-color: var(--surface-card);
}

.sample-stats-dialog .easy-button {
  --el-button-bg-color: var(--surface-card);
  --el-button-text-color: var(--foreground);
  --el-button-border-color: var(--border-default);
  --el-button-hover-bg-color: var(--surface-light);
  --el-button-hover-text-color: var(--accent);
  --el-button-active-bg-color: var(--accent);
  --el-button-active-text-color: var(--surface-card);
}
</style>