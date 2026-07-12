<template>
  <div class="postfix-queue-page">
    <el-card class="glass-card main-card">
      <div class="page-header">
        <div>
          <h1 class="page-title">{{ t('postfix.queueTitle') }}</h1>
          <p class="page-subtitle">{{ t('postfix.queueSubtitle') }}</p>
        </div>
        <div class="header-actions">
          <el-select v-model="selectedAgentId" :placeholder="t('postfix.queueSelectAgent')" class="agent-select" @change="onAgentChange">
            <el-option v-for="agent in onlineAgents" :key="agent.id" :label="`${agent.name} (${agent.host})`" :value="agent.id" />
          </el-select>
          <el-button class="easy-button" :disabled="!selectedAgentId" @click="loadQueue">
            <el-icon><Refresh /></el-icon>
            {{ t('common.refresh') }}
          </el-button>
          <el-button type="primary" class="easy-button" :disabled="!selectedAgentId" @click="flushQueue">
            <el-icon><VideoPlay /></el-icon>
            {{ t('postfix.queueFlushAll') }}
          </el-button>
        </div>
      </div>

      <!-- Stats cards -->
      <div class="stats-grid" v-if="selectedAgentId">
        <div class="stat-card stat--total">
          <div class="stat-icon"><Message /></div>
          <div class="stat-info">
            <span class="stat-value">{{ stats.total }}</span>
            <span class="stat-label">{{ t('postfix.queueTotal') }}</span>
          </div>
        </div>
        <div class="stat-card stat--active">
          <div class="stat-icon"><Cpu /></div>
          <div class="stat-info">
            <span class="stat-value">{{ stats.active }}</span>
            <span class="stat-label">{{ t('postfix.queueActive') }}</span>
          </div>
        </div>
        <div class="stat-card stat--deferred">
          <div class="stat-icon"><Clock /></div>
          <div class="stat-info">
            <span class="stat-value">{{ stats.deferred }}</span>
            <span class="stat-label">{{ t('postfix.queueDeferred') }}</span>
          </div>
        </div>
        <div class="stat-card stat--held">
          <div class="stat-icon"><VideoPause /></div>
          <div class="stat-info">
            <span class="stat-value">{{ stats.held }}</span>
            <span class="stat-label">{{ t('postfix.queueHeld') }}</span>
          </div>
        </div>
      </div>

      <!-- Filter bar -->
      <div class="filter-bar" v-if="selectedAgentId">
        <el-select v-model="filter.status" clearable :placeholder="t('postfix.queueStatus')" class="filter-select" @change="loadQueue">
          <el-option :label="t('postfix.queueAll')" value="" />
          <el-option :label="`${t('postfix.queueActiveLabel')} (*)`" value="active" />
          <el-option :label="`${t('postfix.queueDeferredLabel')} (-)`" value="deferred" />
          <el-option :label="`${t('postfix.queueHeldLabel')} (**)`" value="held" />
        </el-select>
        <el-input v-model="filter.sender" :placeholder="t('postfix.queueSearchPlaceholder')" clearable class="easy-input filter" @clear="loadQueue" @keyup.enter="loadQueue" />
        <el-input v-model="filter.recipient" :placeholder="t('postfix.queueRecipientPlaceholder')" clearable class="easy-input filter" @clear="loadQueue" @keyup.enter="loadQueue" />
        <el-button type="primary" class="easy-button" @click="loadQueue">
          <el-icon><Search /></el-icon>
          {{ t('common.search') }}
        </el-button>
      </div>

      <!-- Action bar -->
      <div class="action-bar" v-if="selectedAgentId && messages.length > 0">
        <span class="selected-count">{{ selectedMessages.length }} / {{ messages.length }} {{ t('common.selected') }}</span>
        <div class="action-buttons">
          <el-button type="danger" class="easy-button" :disabled="selectedMessages.length === 0" @click="deleteSelected">
            <el-icon><Delete /></el-icon>
            {{ t('postfix.queueDeleteSelected') }}
          </el-button>
          <el-button type="warning" class="easy-button" :disabled="selectedMessages.length === 0" @click="resendSelected">
            <el-icon><RefreshRight /></el-icon>
            {{ t('postfix.queueResendSelected') }}
          </el-button>
        </div>
      </div>

      <!-- Message table -->
      <el-table
        v-if="selectedAgentId"
        v-loading="loading"
        :data="messages"
        @selection-change="onSelectionChange"
        class="easy-table queue-table"
        max-height="500"
      >
        <el-table-column type="selection" width="50" fixed />
        <el-table-column prop="queueId" :label="t('postfix.queueId')" min-width="160" />
        <el-table-column :label="t('postfix.queueStatus')" min-width="160">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sender" :label="t('postfix.queueSender')" min-width="160" show-overflow-tooltip />
        <el-table-column :label="t('postfix.queueRecipients')" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.recipients.join(', ') }}</template>
        </el-table-column>
        <el-table-column prop="size" :label="t('postfix.queueSize')" width="160" sortable>
          <template #default="{ row }">{{ formatSize(row.size) }}</template>
        </el-table-column>
        <el-table-column prop="age" :label="t('postfix.queueAge')" width="160" />
        <el-table-column :label="t('common.actions')" width="160">
          <template #default="{ row }">
            <el-button size="small" type="primary" class="easy-button" @click="viewDetail(row)">
                <el-icon :size="18">
                    <View />
                </el-icon>
            </el-button>
            <el-button size="small" type="danger" class="easy-button" @click="deleteMessage(row)">
                <el-icon :size="18">
                    <Delete />
                </el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Empty state -->
      <div v-if="selectedAgentId && !loading && messages.length === 0" class="empty-state">
        <el-icon class="empty-icon"><Message /></el-icon>
        <p>{{ t('postfix.queueNoData') }}</p>
      </div>

      <!-- No agent selected -->
      <div v-if="!selectedAgentId" class="empty-state">
        <el-icon class="empty-icon"><OfficeBuilding /></el-icon>
        <p>{{ t('postfix.queueNoAgent') }}</p>
      </div>

      <!-- Pagination -->
      <div class="pagination-container" v-if="selectedAgentId && total > pageSize">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadQueue"
          class="easy-pagination"
        />
      </div>
    </el-card>

    <!-- Detail drawer -->
    <el-drawer v-model="showDetailDrawer" :title="t('postfix.queueDetail')" size="520px" destroy-on-close class="easy-drawer">
      <template v-if="detailMessage">
        <el-descriptions class="easy-descriptions" :column="1" size="small" label-width="80px" style="border-radius: 8px;">
          <el-descriptions-item :label="t('postfix.queueId')">{{ detailMessage.queueId }}</el-descriptions-item>
          <el-descriptions-item :label="t('postfix.queueStatus')">
            <el-tag :type="statusTagType(detailMessage.status)" size="small">{{ statusLabel(detailMessage.status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('postfix.queueSender')">{{ detailMessage.sender }}</el-descriptions-item>
          <el-descriptions-item :label="t('postfix.queueRecipients')">
            <div class="recipient-list">
              <el-tag v-for="(rec, i) in detailMessage.recipients" :key="i" class="recipient-tag" size="small">{{ rec }}</el-tag>
            </div>
          </el-descriptions-item>
          <el-descriptions-item :label="t('postfix.queueSize')">{{ formatSize(detailMessage.size) }}</el-descriptions-item>
          <el-descriptions-item :label="t('postfix.queueAge')">{{ detailMessage.age }}</el-descriptions-item>
          <el-descriptions-item :label="t('postfix.queueStatusText')">
            <div class="status-text">{{ detailMessage.statusText || '—' }}</div>
          </el-descriptions-item>
        </el-descriptions>
        <div class="detail-actions">
          <el-button type="warning" class="easy-button" @click="resendMessage(detailMessage)">
            <el-icon><RefreshRight /></el-icon>
            {{ t('postfix.queueResendSelected') }}
          </el-button>
          <el-button type="danger" class="easy-button" @click="deleteMessage(detailMessage)">
            <el-icon><Delete /></el-icon>
            {{ t('common.delete') }}
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Message, Refresh, VideoPlay, Cpu, Clock, VideoPause,
  Search, Delete, RefreshRight, OfficeBuilding
} from '@element-plus/icons-vue'
import { postfixApi, type PostfixAgent, type QueueMessage, type QueueStats } from '../api/postfix'

const { t } = useI18n()

const loading = ref(false)
const agents = ref<PostfixAgent[]>([])
const onlineAgents = computed(() => agents.value.filter(a => a.lastStatus === 'online' || a.enabled))

const selectedAgentId = ref('')
const stats = ref<QueueStats>({ total: 0, active: 0, deferred: 0, held: 0 })

const filter = ref({ status: '', sender: '', recipient: '' })
const messages = ref<QueueMessage[]>([])
const selectedMessages = ref<QueueMessage[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)

const showDetailDrawer = ref(false)
const detailMessage = ref<QueueMessage | null>(null)
const detailAgentId = ref('')

onMounted(async () => {
  await loadAgents()
})

async function loadAgents() {
  try {
    const res = await postfixApi.listAgents({ page: 1, pageSize: 100 })
    agents.value = res.data
    // Auto-select first online agent
    if (onlineAgents.value.length > 0 && !selectedAgentId.value) {
      selectedAgentId.value = onlineAgents.value[0].id
      await loadQueue()
    }
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.loadFailed'))
  }
}

async function loadQueue() {
  if (!selectedAgentId.value) return
  loading.value = true
  try {
    const [listRes, statsRes] = await Promise.all([
      postfixApi.listQueue(selectedAgentId.value, {
        status: filter.value.status || undefined,
        sender: filter.value.sender || undefined,
        recipient: filter.value.recipient || undefined,
        page: page.value,
        pageSize: pageSize.value
      }),
      postfixApi.getQueueStats(selectedAgentId.value)
    ])
    messages.value = listRes.data.messages || []
    total.value = listRes.data.total || 0
    stats.value = statsRes.data
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('postfix.queueOperationFailed'))
  } finally {
    loading.value = false
  }
}

function onAgentChange() {
  page.value = 1
  loadQueue()
}

function onSelectionChange(rows: QueueMessage[]) {
  selectedMessages.value = rows
}

function statusTagType(status: string) {
  switch (status) {
    case 'active': return ''
    case 'deferred': return 'warning'
    case 'held': return 'danger'
    default: return 'info'
  }
}

const statusLabel = (status: string) => {
  const map: Record<string, string> = {
    active: t('postfix.queueActiveLabel'),
    deferred: t('postfix.queueDeferredLabel'),
    held: t('postfix.queueHeldLabel')
  }
  return map[status] || status
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function viewDetail(msg: QueueMessage) {
  detailMessage.value = msg
  detailAgentId.value = selectedAgentId.value
  showDetailDrawer.value = true
}

async function deleteMessage(msg: QueueMessage) {
  try {
    await ElMessageBox.confirm(t('postfix.queueDeleteConfirm', { count: 1 }), t('common.delete'), { type: 'warning' })
    await postfixApi.deleteQueueMessages(detailAgentId.value, [msg.queueId])
    ElMessage.success(t('postfix.queueDeleteSuccess', { count: 1 }))
    showDetailDrawer.value = false
    await loadQueue()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || t('postfix.queueOperationFailed'))
    }
  }
}

async function resendMessage(msg: QueueMessage) {
  try {
    await postfixApi.resendQueueMessages(detailAgentId.value, [msg.queueId])
    ElMessage.success(t('postfix.queueResendSuccess', { count: 1 }))
    showDetailDrawer.value = false
    await loadQueue()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('postfix.queueOperationFailed'))
  }
}

async function deleteSelected() {
  if (selectedMessages.value.length === 0) return
  try {
    await ElMessageBox.confirm(t('postfix.queueDeleteConfirm', { count: selectedMessages.value.length }), t('common.delete'), { type: 'warning' })
    const ids = selectedMessages.value.map(m => m.queueId)
    await postfixApi.deleteQueueMessages(selectedAgentId.value, ids)
    ElMessage.success(t('postfix.queueDeleteSuccess', { count: ids.length }))
    await loadQueue()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || t('postfix.queueOperationFailed'))
    }
  }
}

async function resendSelected() {
  if (selectedMessages.value.length === 0) return
  try {
    await ElMessageBox.confirm(t('postfix.queueResendConfirm', { count: selectedMessages.value.length }), t('common.confirm'), { type: 'warning' })
    const ids = selectedMessages.value.map(m => m.queueId)
    await postfixApi.resendQueueMessages(selectedAgentId.value, ids)
    ElMessage.success(t('postfix.queueResendSuccess', { count: ids.length }))
    await loadQueue()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || t('postfix.queueOperationFailed'))
    }
  }
}

async function flushQueue() {
  if (!selectedAgentId.value) return
  try {
    await ElMessageBox.confirm(t('postfix.queueFlushConfirm'), t('common.confirm'), { type: 'warning' })
    await postfixApi.flushQueue(selectedAgentId.value)
    ElMessage.success(t('postfix.queueFlushSuccess'))
    await loadQueue()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || t('postfix.queueOperationFailed'))
    }
  }
}
</script>

<style scoped>
.postfix-queue-page {
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

.page-title { font-size: 24px; font-weight: 700; color: var(--foreground); margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.page-subtitle { font-size: 14px; color: var(--foreground-muted); margin: 4px 0 0 0; }
.header-actions { display: flex; gap: 10px; align-items: center; }
.agent-select { width: 280px; }

/* Stats */
.stats-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 20px; }
.stat-card { display: flex; align-items: center; gap: 12px; background: var(--surface); border-radius: 12px; padding: 16px; border: 1px solid var(--border-default); }
.stat--total { border-left: 3px solid var(--primary); }
.stat--active { border-left: 3px solid var(--success); }
.stat--deferred { border-left: 3px solid var(--warning); }
.stat--held { border-left: 3px solid var(--danger); }
.stat-icon { font-size: 28px; color: var(--foreground-muted); }
.stat-info { display: flex; flex-direction: column; }
.stat-value { font-size: 24px; font-weight: 700; color: var(--foreground); }
.stat-label { font-size: 13px; color: var(--foreground-muted); }

/* Filter */
.filter-bar { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.filter-select { width: 160px; }

.filter {
    width: 220px;
    flex: 0 0 220px;
}
.filter .el-input__wrapper {
    background-color: transparent !important;
}

/* Action bar */
.action-bar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; padding: 10px 16px; background: var(--surface-hover); border-radius: 8px; }
.selected-count { font-size: 14px; color: var(--foreground-muted); }
.action-buttons { display: flex; gap: 8px; }

/* Empty state */
.empty-state { text-align: center; padding: 60px 20px; color: var(--foreground-muted); }
.empty-icon { font-size: 56px; margin-bottom: 16px; color: var(--border-default); }

/* Pagination */
.pagination-container { margin-top: 16px; display: flex; justify-content: center; }

/* Detail drawer */
.easy-drawer :deep(.el-drawer__header) {
  margin-bottom: 0;
  padding: 20px 24px 16px;
  border-bottom: 1px solid var(--border-default);
}

.easy-drawer :deep(.el-drawer__body) {
  padding: 20px 24px;
}

.easy-descriptions {
  margin-bottom: 20px;
}

.easy-descriptions :deep(.el-descriptions__label) {
  font-weight: 600;
  color: var(--foreground-muted);
  width: 80px;
}

.easy-descriptions :deep(.el-descriptions__content) {
  color: var(--foreground);
}

.recipient-list { display: flex; flex-wrap: wrap; gap: 6px; }
.recipient-tag { margin: 0; }
.status-text { white-space: pre-wrap; max-height: 200px; overflow-y: auto; font-size: 13px; color: var(--foreground-muted); }
.detail-actions { display: flex; justify-content: center; gap: 10px; margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--border-default); }
</style>
