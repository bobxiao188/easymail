<template>
  <div class="postfix-status-page">
    <el-card class="glass-card main-card">
      <div class="page-header">
        <div>
          <h1 class="page-title">{{ t('postfix.statusTitle') }}</h1>
          <p class="page-subtitle">{{ t('postfix.statusSubtitle') }}</p>
        </div>
        <div class="header-actions">
          <el-button class="easy-button" @click="loadSummary">
            <el-icon><Refresh /></el-icon>
            {{ t('common.refresh') }}
          </el-button>
        </div>
      </div>

      <!-- Status summary cards -->
      <div class="summary-cards">
        <el-card class="glass-card main-card stat-card">
          <div class="stat-value">{{ totalAgents }}</div>
          <div class="stat-label">{{ t('postfix.totalAgents') }}</div>
        </el-card>
        <el-card class="glass-card main-card stat-card">
          <div class="stat-value online">{{ onlineAgents }}</div>
          <div class="stat-label">{{ t('postfix.onlineAgents') }}</div>
        </el-card>
        <el-card class="glass-card main-card stat-card">
          <div class="stat-value">{{ totalConfigs }}</div>
          <div class="stat-label">{{ t('postfix.totalConfigs') }}</div>
        </el-card>
      </div>

      <!-- Agent status table -->
      <el-card class="glass-card main-card status-table-card">
        <template #header>
          <span class="table-title">{{ t('postfix.agentStatusTable') }}</span>
        </template>
        <el-table :data="agents" stripe class="easy-table">
          <el-table-column prop="agentName" :label="t('postfix.agentName')" min-width="120" />
          <el-table-column prop="host" :label="t('postfix.agentHost')" min-width="120" />
          <el-table-column :label="t('postfix.online')">
            <template #default="{ row }">
              <el-tag :type="row.online ? 'success' : 'danger'" size="small">
                {{ row.online ? t('common.yes') : t('common.no') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="lastSyncAt" :label="t('postfix.lastSync')" >
            <template #default="{ row }">
              <span class="status-row-value">{{ formatDateTime(row.lastSyncAt) || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('postfix.configHash')" >
            <template #default="{ row }">
              <code class="hash-text">{{ row.configHash?.slice(0, 12) || '-' }}</code>
            </template>
          </el-table-column>
          <el-table-column :label="t('postfix.upToDate')">
            <template #default="{ row }">
              <el-tag :type="row.upToDate ? 'success' : 'warning'" size="small">
                {{ row.upToDate ? t('common.yes') : t('common.no') }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        <div v-if="agents.length === 0" class="table-empty">
          <p>{{ t('postfix.noAgentsStatus') }}</p>
        </div>
      </el-card>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { postfixApi, type AgentConfigStatus } from '../api/postfix'
import { formatDateTime } from '../utils/times'

const { t } = useI18n()
const loading = ref(false)
const agents = ref<AgentConfigStatus[]>([])
const totalConfigs = ref(0)

onMounted(() => loadSummary())

const totalAgents = computed(() => (agents.value || []).length)
const onlineAgents = computed(() => (agents.value || []).filter(a => a.online).length)

async function loadSummary() {
  loading.value = true
  try {
    const [statusRes, configsRes] = await Promise.all([
      postfixApi.status(),
      postfixApi.listConfigs({ pageSize: 1 })
    ])
    agents.value = statusRes.data?.agents || []
    totalConfigs.value = configsRes.meta.total
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.postfix-status-page {
  height: 100%;
  margin: 0 auto;
}

.main-card {
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
.summary-cards { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 20px; }
.stat-card { text-align: center; padding: 20px; }
.stat-value { font-size: 36px; font-weight: 700; color: var(--foreground); }
.stat-value.online { color: var(--success-color); }
.stat-label { font-size: 14px; color: var(--foreground-muted); margin-top: 8px; }
.table-title { font-size: 16px; font-weight: 700; color: var(--foreground); margin: 0; }
.hash-text { font-size: 12px; background: var(--surface-hover); padding: 2px 6px; border-radius: 4px; }
.table-empty { text-align: center; padding: 20px; color: var(--foreground-muted); }
</style>