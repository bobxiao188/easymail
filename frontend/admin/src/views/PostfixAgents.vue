<template>
  <div class="postfix-agents-page">
    <el-card class="glass-card main-card">
      <div class="page-header">
        <div>
          <h1 class="page-title">{{ t('postfix.agentsTitle') }}</h1>
          <p class="page-subtitle">{{ t('postfix.agentsSubtitle') }}</p>
        </div>
        <div class="header-actions">
          <el-button class="easy-button" @click="loadAgents">
            <el-icon><Refresh /></el-icon>
            {{ t('common.refresh') }}
          </el-button>
          <el-button type="primary" class="easy-button" @click="showCreateDialog = true">
            <el-icon><Plus /></el-icon>
            {{ t('common.add') }}
          </el-button>
        </div>
      </div>

      <!-- Agent cards -->
      <div class="agent-grid" v-loading="loading">
        <div v-for="agent in agents" :key="agent.id" class="agent-card" :class="`agent--${agent.lastStatus}`">
          <div class="card-header">
            <div class="card-info">
              <h3 class="card-info-title">{{ agent.name }}</h3>
              <span class="agent-host">{{ agent.host }}</span>
            </div>
            <el-tag class="status-tag" :type="statusTagType(agent.lastStatus)">
              {{ agent.lastStatus }}
            </el-tag>
          </div>
          <div class="card-body">
            <div class="status-row">
              <span class="status-row-label">{{ t('postfix.lastSync') }}:</span>
              <span class="status-row-value">{{ formatDateTime(agent.lastSyncAt) || '-' }}</span>
            </div>
            <div class="status-row" v-if="agent.description">
              <span class="status-row-label">{{ t('common.description') }}:</span>
              <span class="status-row-value">{{ agent.description }}</span>
            </div>
          </div>
          <div class="card-actions">
            <el-button size="small" class="easy-button" @click="checkStatus(agent)">{{ t('postfix.checkStatus') }}</el-button>
            <el-button size="small" type="primary" class="easy-button" @click="deployConfig(agent)">{{ t('postfix.deploy') }}</el-button>
            <el-button size="small" class="easy-button" @click="editAgent(agent)">{{ t('common.edit') }}</el-button>
            <el-button size="small" type="danger" class="easy-button" @click="deleteAgent(agent)">{{ t('common.delete') }}</el-button>
          </div>
        </div>
        <div v-if="agents.length === 0" class="empty-state">
          <el-icon class="empty-icon"><Cpu /></el-icon>
          <p>{{ t('postfix.noAgents') }}</p>
        </div>
      </div>

      <div class="pagination-container">
        <el-pagination
          v-if="total > pageSize"
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          @current-change="loadAgents"
          class="easy-pagination"
        />
      </div>
    </el-card>

    <!-- Create/Edit dialog -->
    <el-dialog v-model="showCreateDialog" :title="isEditing ? t('common.edit') : t('common.add')" width="53%" class="easy-dialog">
      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('postfix.agentName')" required>
          <el-input v-model="form.name" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('postfix.agentHost')" required>
          <el-input v-model="form.host" placeholder="192.168.1.10:8081" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('postfix.agentToken')" :required="!isEditing">
          <el-input v-model="form.token" type="password" show-password class="easy-input" />
          <span v-if="isEditing" class="form-hint">{{ t('postfix.tokenHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('common.description')">
          <el-input v-model="form.description" type="textarea" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('common.enabled')" v-if="isEditing">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="showCreateDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" class="easy-button" @click="saveAgent" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Deploy dialog -->
    <el-dialog v-model="showDeployDialog" :title="t('postfix.deployConfig')" width="53%" class="easy-dialog">
      <div v-loading="deploying">
        <el-alert v-if="deployResult" :title="deployResult" :type="deploySuccess ? 'success' : 'error'" show-icon class="deploy-alert" />
        <div class="deploy-actions">
          <el-button class="easy-button" @click="pushConfig(selectedAgent!)" :disabled="deploying">
            {{ t('postfix.pushOnly') }}
          </el-button>
          <el-button type="primary" class="easy-button" @click="pushAndApply(selectedAgent!)" :disabled="deploying">
            {{ t('postfix.pushAndApply') }}
          </el-button>
          <el-button class="easy-button" @click="rollbackConfig(selectedAgent!)" :disabled="deploying">
            {{ t('postfix.rollback') }}
          </el-button>
        </div>

        <el-divider />

        <h4>{{ t('postfix.recentLogs') }}</h4>
        <el-table :data="deployLogs" size="small" max-height="300" class="easy-table">
          <el-table-column prop="action" label="Action" width="120" />
          <el-table-column prop="status" label="Status">
            <template #default="{ row }">
              <el-tag class="status-tag" :type="row.status === 'success' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Time">
            <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
          </el-table-column>
          <el-table-column prop="errorMessage" label="Error" show-overflow-tooltip />
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Cpu } from '@element-plus/icons-vue'
import { postfixApi, type PostfixAgent, type DeliveryLog } from '../api/postfix'
import { formatDateTime } from '../utils/times'

const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const agents = ref<PostfixAgent[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const showCreateDialog = ref(false)
const isEditing = ref(false)
const editingId = ref('')
const form = ref({ name: '', host: '', token: '', description: '', enabled: true })

const showDeployDialog = ref(false)
const selectedAgent = ref<PostfixAgent | null>(null)
const deploying = ref(false)
const deployResult = ref('')
const deploySuccess = ref(false)
const deployLogs = ref<DeliveryLog[]>([])

onMounted(() => loadAgents())

function statusTagType(status: string) {
  switch (status) {
    case 'online': return 'success'
    case 'offline': return 'danger'
    case 'error': return 'warning'
    default: return 'info'
  }
}

async function loadAgents() {
  loading.value = true
  try {
    const res = await postfixApi.listAgents({ page: page.value, pageSize: pageSize.value })
    agents.value = res.data
    total.value = res.meta.total
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function checkStatus(agent: PostfixAgent) {
  try {
    const res = await postfixApi.checkAgentStatus(agent.id)
    const s = res.data
    ElMessage.info(`Postfix: ${s.postfixRunning ? 'Running' : 'Stopped'} | Hash: ${s.configHash?.slice(0, 12) || '-'}`)
    await loadAgents()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || 'Agent unreachable')
  }
}

function editAgent(agent: PostfixAgent) {
  isEditing.value = true
  editingId.value = agent.id
  form.value = { name: agent.name, host: agent.host, token: '', description: agent.description || '', enabled: agent.enabled }
  showCreateDialog.value = true
}

async function saveAgent() {
  saving.value = true
  try {
    if (isEditing.value) {
      await postfixApi.updateAgent(editingId.value, form.value)
    } else {
      await postfixApi.createAgent(form.value)
    }
    ElMessage.success(t('common.saveSuccess'))
    showCreateDialog.value = false
    resetForm()
    await loadAgents()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.operationFailed'))
  } finally {
    saving.value = false
  }
}

async function deleteAgent(agent: PostfixAgent) {
  try {
    await ElMessageBox.confirm(t('common.deleteConfirm'), t('common.delete'), { type: 'warning' })
    await postfixApi.deleteAgent(agent.id)
    ElMessage.success(t('common.deleteSuccess'))
    await loadAgents()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || t('common.operationFailed'))
    }
  }
}

function resetForm() {
  form.value = { name: '', host: '', token: '', description: '', enabled: true }
  isEditing.value = false
  editingId.value = ''
}

async function deployConfig(agent: PostfixAgent) {
  selectedAgent.value = agent
  deployResult.value = ''
  deploySuccess.value = false
  deployLogs.value = []
  showDeployDialog.value = true
  await loadLogs(agent.id)
}

async function pushConfig(agent: PostfixAgent) {
  deploying.value = true
  deployResult.value = ''
  try {
    await postfixApi.pushToAgent(agent.id)
    deployResult.value = 'Config pushed to agent successfully'
    deploySuccess.value = true
    await loadLogs(agent.id)
  } catch (e: any) {
    deployResult.value = e?.response?.data?.message || 'Push failed'
    deploySuccess.value = false
  } finally {
    deploying.value = false
  }
}

async function pushAndApply(agent: PostfixAgent) {
  deploying.value = true
  deployResult.value = ''
  try {
    await postfixApi.pushAndApply(agent.id)
    deployResult.value = 'Config pushed and applied successfully'
    deploySuccess.value = true
    await loadLogs(agent.id)
    await checkStatus(agent)
  } catch (e: any) {
    deployResult.value = e?.response?.data?.message || 'Deploy failed'
    deploySuccess.value = false
  } finally {
    deploying.value = false
  }
}

async function rollbackConfig(agent: PostfixAgent) {
  deploying.value = true
  deployResult.value = ''
  try {
    await postfixApi.rollbackOnAgent(agent.id)
    deployResult.value = 'Config rolled back successfully'
    deploySuccess.value = true
    await loadLogs(agent.id)
  } catch (e: any) {
    deployResult.value = e?.response?.data?.message || 'Rollback failed'
    deploySuccess.value = false
  } finally {
    deploying.value = false
  }
}

async function loadLogs(agentId: string) {
  try {
    const res = await postfixApi.listLogs(agentId, { limit: 10 })
    deployLogs.value = res.data
  } catch (_) { /* ignore */ }
}
</script>

<style scoped>
.postfix-agents-page {
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
.header-actions { display: flex; gap: 10px; }
.agent-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 16px; }
.agent-card { background: var(--surface); border-radius: 12px; padding: 16px; border: 1px solid var(--border-default); }
.agent--online { border-color: var(--success); }
.agent--offline { border-color: var(--danger); }
.agent--error { border-color: var(--warning); }
.card-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; }
.card-info-title { margin: 0; font-size: 16px; font-weight: 700; color: var(--foreground); }
.agent-host { font-size: 12px; color: var(--foreground-muted); }
.card-body { margin-bottom: 12px; }
.status-row { display: flex; gap: 8px; font-size: 13px; margin-bottom: 4px; }
.status-row span:first-child { color: var(--foreground-muted); min-width: 80px; }
.card-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.empty-state { grid-column: 1 / -1; text-align: center; padding: 40px; color: var(--foreground-muted); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.pagination-container { margin-top: 16px; display: flex; justify-content: center; }
.form-hint { font-size: 12px; color: var(--foreground-muted); }
.deploy-actions { display: flex; gap: 10px; margin-bottom: 16px; }
.deploy-alert { margin-bottom: 16px; background-color: var(--surface-hover); border: 1px solid var(--border-color); border-radius: 8px; padding: 16px; color: var(--warning-color); }
.status-tag { font-size: 12px; font-weight: 500; color: var(--success-color); }
.status-row-label { color: var(--foreground-muted); min-width: 80px; }
.status-row-value { color: var(--foreground); }
</style>