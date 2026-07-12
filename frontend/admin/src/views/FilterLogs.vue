<template>
  <div class="scanner-logs-page">
    <el-card class="glass-card main-card">
      <div class="search-bar scanner-logs-toolbar">
        <div class="search-bar-left scanner-logs-toolbar-left">
          <el-input
            v-model="filterIp"
            class="easy-input"
            :placeholder="t('filterLogs.ipPlaceholder')"
            clearable
            style="width: 9.5rem"
            @keyup.enter="applySearch"
          />
          <el-input
            v-model="filterSender"
            class="easy-input"
            :placeholder="t('filterLogs.senderPlaceholder')"
            clearable
            style="min-width: 10rem; width: 11rem"
            @keyup.enter="applySearch"
          />
          <el-input
            v-model="filterRcpt"
            class="easy-input"
            :placeholder="t('filterLogs.rcptPlaceholder')"
            clearable
            style="min-width: 10rem; width: 11rem"
            @keyup.enter="applySearch"
          />
          <el-date-picker
            v-model="createdRange"
            type="datetimerange"
            :range-separator="t('filterLogs.rangeSeparator')"
            :start-placeholder="t('filterLogs.startTime')"
            :end-placeholder="t('filterLogs.endTime')"
            class="easy-input scanner-logs-date-range"
            :teleported="true"
          />
          <el-button type="primary" class="easy-button" @click="applySearch">{{ t('filterLogs.query') }}</el-button>
          <el-button class="easy-button" @click="resetFilters">{{ t('filterLogs.resetQuery') }}</el-button>
        </div>
        <div class="search-bar-right">
          <el-button type="primary" class="easy-button" @click="loadLogs">
            <el-icon><Refresh /></el-icon>
            {{ t('common.refresh') }}
          </el-button>
        </div>
      </div>

      <el-table v-loading="loading" :data="rows" class="easy-table" stripe>
        <el-table-column prop="id" :label="t('common.id')" width="120" />
        <el-table-column prop="ip" :label="t('filterLogs.ip')" min-width="130" show-overflow-tooltip />
        <el-table-column prop="sender" :label="t('filterLogs.sender')" min-width="160" show-overflow-tooltip />
        <el-table-column prop="recipient" :label="t('filterLogs.recipient')" min-width="160" show-overflow-tooltip />
        <el-table-column prop="subject" :label="t('filterLogs.subject')" min-width="160" show-overflow-tooltip />
        <el-table-column prop="actionApplied" :label="t('filterLogs.action')" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="actionTagType(row.actionApplied)">{{ row.actionApplied }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ruleId" :label="t('filterLogs.ruleId')" width="120">
          <template #default="{ row }">{{ row.ruleId ?? '—' }}</template>
        </el-table-column>
        <el-table-column prop="queueId" :label="t('filterLogs.queueId')" min-width="120" show-overflow-tooltip />
        <el-table-column prop="durationMs" :label="t('filterLogs.duration')" width="120" />
        <el-table-column prop="createdAt" :label="t('filterLogs.createdAt')" width="220">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.action')" width="100">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button class="easy-icon-button" type="primary" size="small" @click.stop="openDetail(row)">
                <el-icon><View /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-container">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          class="easy-pagination"
          @size-change="loadLogs"
          @current-change="loadLogs"
        />
      </div>
    </el-card>

    <el-drawer v-model="drawerVisible" :title="t('filterLogs.logDetail')" size="520px" destroy-on-close class="easy-drawer">
      <template v-if="detail">
        <el-descriptions class="easy-descriptions" :column="1" size="small" label-width="80px" style="border-radius: 8px;">
          <el-descriptions-item :label="t('common.id')">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.traceId')">{{ detail.traceId }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.stage')">{{ formatPipelineStageLabel(t, detail.stage) }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.ip')">{{ detail.ip || '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.queueId')">{{ detail.queueId || '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.sender')">{{ detail.sender || '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.recipient')">{{ detail.recipient }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.subject')">{{ detail.subject || '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.ruleId')">{{ detail.ruleId ?? '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.action')">{{ detail.actionApplied }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.duration')">{{ detail.durationMs }}</el-descriptions-item>
          <el-descriptions-item :label="t('filterLogs.createdAt')">{{ formatDateTime(detail.createdAt) }}</el-descriptions-item>
        </el-descriptions>
        <div class="block-container">
          <h4 class="block-title">{{ t('filterLogs.featureSnapshot') }}</h4>
          <el-button type="primary" size="small" @click="copyJson(detail.featureSnapshotJson)">
            {{ t('common.copy') }}
          </el-button>
        </div>
        <pre class="json-block">{{ prettyJson(detail.featureSnapshotJson) }}</pre>
        <h4 class="block-title">{{ t('filterLogs.conditionTrace') }}</h4>
        <pre class="json-block">{{ prettyJson(detail.conditionTraceJson) }}</pre>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { scannerApi, type ScannerScannerLog } from '../api/filter'
import { formatPipelineStageLabel } from '../utils/filterStage'
import { formatDateTime } from '../utils/times'
import { View } from '@element-plus/icons-vue'

const { t } = useI18n()

const loading = ref(false)
const rows = ref<ScannerScannerLog[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const filterIp = ref('')
const filterSender = ref('')
const filterRcpt = ref('')
const createdRange = ref<[Date, Date] | null>(null)

const drawerVisible = ref(false)
const detail = ref<ScannerScannerLog | null>(null)

function actionTagType(a: string) {
  if (a === 'spam') return 'warning'
  if (a === 'quarantine') return 'danger'
  return 'success'
}

function prettyJson(raw: string) {
  if (!raw || !raw.trim()) return '—'
  try {
    const o = JSON.parse(raw)
    return JSON.stringify(o, null, 2)
  } catch {
    return raw
  }
}

function deliveryLogQueryParams() {
  const params: {
    page: number
    pageSize: number
    ip?: string
    sender?: string
    rcpt?: string
    created_from?: string
    created_to?: string
  } = {
    page: page.value,
    pageSize: pageSize.value
  }
  const ip = filterIp.value.trim()
  const sender = filterSender.value.trim()
  const rcpt = filterRcpt.value.trim()
  if (ip) params.ip = ip
  if (sender) params.sender = sender
  if (rcpt) params.rcpt = rcpt
  if (createdRange.value?.[0]) {
    params.created_from = createdRange.value[0].toISOString()
  }
  if (createdRange.value?.[1]) {
    params.created_to = createdRange.value[1].toISOString()
  }
  return params
}

function applySearch() {
  page.value = 1
  void loadLogs()
}

function resetFilters() {
  filterIp.value = ''
  filterSender.value = ''
  filterRcpt.value = ''
  createdRange.value = null
  page.value = 1
  void loadLogs()
}

async function loadLogs() {
  loading.value = true
  try {
    const res = await scannerApi.listScannerLogs(deliveryLogQueryParams())
    if (res.code === 0 && Array.isArray(res.data)) {
      rows.value = res.data
      if (res.meta) {
        total.value = res.meta.total ?? 0
      }
    } else {
      ElMessage.error(res.message || t('filterLogs.loadFailed'))
    }
  } catch {
    ElMessage.error(t('filterLogs.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function openDetail(row: ScannerScannerLog) {
  try {
    const res = await scannerApi.getScannerLog(row.id)
    if (res.code === 0 && res.data) {
      detail.value = res.data
      drawerVisible.value = true
    } else {
      detail.value = row
      drawerVisible.value = true
    }
  } catch {
    detail.value = row
    drawerVisible.value = true
  }
}

async function copyJson(json: string) {
  try {
    await navigator.clipboard.writeText(json)
    ElMessage.success(t('filterLogs.copySuccess'))
  } catch {
    ElMessage.error(t('filterLogs.copyFailed'))
  }
}

onMounted(() => {
  void loadLogs()
})
</script>

<style scoped>
.scanner-logs-page {
  height: 100%;
  margin: 0 auto;
}

.main-card {
  min-height: calc(100vh - 140px);
  border: none;
  box-shadow: var(--shadow-card);
}

.main-card :deep(.el-card__body) {
  padding: 20px 25px 28px;
}

.scanner-logs-toolbar {
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-default);
}

.scanner-logs-toolbar-left {
  flex-wrap: wrap;
  gap: 10px 8px;
  align-items: center;
}

.scanner-logs-date-range {
  min-width: min(100%, 18rem);
  max-width: 26rem;
  flex: 1 1 18rem;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  width: 40px;
  height: 40px;
  background: var(--accent-10);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--foreground);
}

.easy-table :deep(.el-table__row) {
  cursor: pointer;
}

.block-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.block-title {
  margin: 20px 0 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--foreground);
}

.json-block {
  margin: 0;
  padding: 12px;
  border-radius: 8px;
  background: var(--surface);
  border: 1px solid var(--border-default);
  font-size: 12px;
  line-height: 1.45;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: ui-monospace, monospace;
}

.detail-desc {
  margin-bottom: 8px;
}

</style>
