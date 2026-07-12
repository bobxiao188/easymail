<template>
  <div class="postfix-configs-page">
    <el-card class="glass-card main-card">
      <div class="page-header">
        <div>
          <h1 class="page-title">{{ t('postfix.configTitle') }}</h1>
          <p class="page-subtitle">{{ t('postfix.configSubtitle') }}</p>
        </div>
      </div>

      <!-- Compact top section: Settings + Alert + Search -->
      <div class="compact-top-section">
        <!-- Search bar -->
        <div class="search-bar">
          <div class="search-bar-left">
            <el-input v-model="searchQuery" :placeholder="t('common.search')" class="easy-input" clearable
              style="width: 12rem;" size="small" @keyup.enter="loadConfigs" />
            <el-button type="primary" size="small" class="easy-button" @click="loadConfigs">{{ t('common.search')
            }}</el-button>
          </div>
          <div class="search-bar-right">
            <div class="settings-inline">
              <span class="settings-label">{{ t('postfix.easymailHost') }}:</span>
              <el-select v-model="settings.easymailHost" class="settings-select" :placeholder="t('postfix.selectIP')">
                <el-option v-for="ip in localIPs" :key="ip" :label="ip" :value="ip" />
              </el-select>
              <el-button type="primary" size="small" class="easy-button" @click="saveSettings"
                :loading="savingSettings">
                <el-icon>
                  <Check />
                </el-icon>
                {{ t('common.save') }}
              </el-button>
            </div>
            <el-button size="small" class="easy-button" @click="previewConfig">
              <el-icon>
                <Document />
              </el-icon>
              {{ t('postfix.preview') }}
            </el-button>
            <el-button type="primary" size="small" class="easy-button" @click="showCreateDialog = true">
              <el-icon>
                <Plus />
              </el-icon>
              {{ t('common.add') }}
            </el-button>
          </div>
        </div>
      </div>

      <!-- Config table -->
      <el-table v-loading="loading" :data="configs" stripe class="easy-table">
        <el-table-column prop="paramName" :label="t('postfix.paramName')" min-width="200" />
        <el-table-column prop="paramValue" :label="t('postfix.paramValue')" min-width="250" show-overflow-tooltip />
        <el-table-column prop="description" :label="t('common.description')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="isManaged" :label="t('postfix.isManaged')" min-width="100">
          <template #default="{ row }">
            {{ row.isManaged ? t('common.yes') : t('common.no') }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.action')" width="160">
          <template #default="{ row }">
            <el-button size="small" link @click="editConfig(row)" v-if="!row.isManaged">
              <el-icon :size="18">
                <Edit />
              </el-icon>
            </el-button>
            <el-button size="small" type="danger" link @click="deleteConfig(row)" v-if="!row.isManaged">
              <el-icon :size="18">
                <Delete />
              </el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-container">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :page-sizes="[10, 15, 50]"
          layout="total, sizes, prev, pager, next, jumper" :total="total" @size-change="handleSizeChange"
          @current-change="handleCurrentChange" class="easy-pagination" />
      </div>
    </el-card>

    <!-- Create / Edit dialog -->
    <el-dialog v-model="showCreateDialog" :title="isEditing ? t('common.edit') : t('common.add')" width="53%"
      class="easy-dialog" @open="onDialogOpen">
      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('postfix.paramName')" required>
          <el-input v-model="form.paramName" :disabled="isEditing" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('postfix.paramValue')" required>
          <el-input v-model="form.paramValue" type="textarea" :rows="3" class="easy-input" />
          <div class="variables-hint" v-if="Object.keys(variables).length > 0">
            <div class="hint-title">{{ t('postfix.availableVariables') }}</div>
            <div class="hint-tags">
              <el-tag v-for="(value, key) in variables" :key="key" size="small" class="var-tag"
                @click="insertVariable(key)" :title="value + ' (click to insert)'">
                ${{ key }}
              </el-tag>
            </div>
          </div>
        </el-form-item>
        <el-form-item :label="t('common.description')">
          <el-input v-model="form.description" class="easy-input" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="showCreateDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" class="easy-button" @click="saveConfig" :loading="saving">{{ t('common.save')
        }}</el-button>
      </template>
    </el-dialog>

    <!-- Config preview dialog -->
    <el-dialog v-model="showPreview" :title="t('postfix.preview')" width="53%" top="5vh" class="easy-dialog">
      <pre class="config-preview">{{ preview?.mainCf }}</pre>
      <template #footer>
        <span class="preview-meta">Hash: {{ preview?.configHash }} | Domains: {{ preview?.domainCount }}</span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Document, Check } from '@element-plus/icons-vue'
import { postfixApi, type PostfixConfig, type ConfigPreview, type PostfixSettings } from '../api/postfix'

const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const savingSettings = ref(false)
const configs = ref<PostfixConfig[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(15)
const searchQuery = ref('')
const showCreateDialog = ref(false)
const isEditing = ref(false)
const editingId = ref('')
const form = ref({ paramName: '', paramValue: '', description: '' })
const showPreview = ref(false)
const preview = ref<ConfigPreview | null>(null)

// Settings
const settings = ref<PostfixSettings>({ easymailHost: '127.0.0.1' })

// Local IP addresses
const localIPs = ref<string[]>(['127.0.0.1'])

// Available variables
const variables = ref<Record<string, string>>({})

onMounted(() => {
  loadConfigs()
  loadSettings()
  loadLocalIPs()
  loadVariables()
})

async function loadSettings() {
  try {
    const res = await postfixApi.getSettings()
    if (res.data) {
      settings.value = res.data
    }
  } catch (e: any) {
    // Fallback to default
    settings.value = { easymailHost: '127.0.0.1' }
  }
}

async function loadLocalIPs() {
  try {
    const res = await postfixApi.getLocalIPs()
    if (res.data && res.data.length > 0) {
      // Filter out empty strings from the response
      localIPs.value = res.data.filter((ip: string) => ip && ip.trim() !== '')
      // If current setting is not in the list and is valid, add it
      if (
        settings.value.easymailHost &&
        settings.value.easymailHost.trim() !== '' &&
        !localIPs.value.includes(settings.value.easymailHost)
      ) {
        localIPs.value.push(settings.value.easymailHost)
      }
    }
  } catch (e: any) {
    // Fallback to default
    localIPs.value = ['127.0.0.1']
  }
}

async function loadVariables() {
  try {
    const res = await postfixApi.getVariables()
    if (res.data) {
      variables.value = res.data
    }
  } catch (e: any) {
    variables.value = {}
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    await postfixApi.updateSettings(settings.value)
    ElMessage.success(t('common.saveSuccess'))
    // Reload variables with new host
    await loadVariables()
    // Reload local IPs to ensure the current setting is in the list
    await loadLocalIPs()
    // Reload configs because __postfix_settings__ changes may affect other params
    await loadConfigs()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.operationFailed'))
  } finally {
    savingSettings.value = false
  }
}

function onDialogOpen() {
  // Reload variables when dialog opens
  loadVariables()
}

function insertVariable(varName: string) {
  form.value.paramValue += '$' + varName
}

async function loadConfigs() {
  loading.value = true
  try {
    const res = await postfixApi.listConfigs({
      keyword: searchQuery.value,
      page: page.value,
      pageSize: pageSize.value
    })
    configs.value = res.data
    total.value = res.meta.total
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function handleSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  loadConfigs()
}

function handleCurrentChange(current: number) {
  page.value = current
  loadConfigs()
}

async function previewConfig() {
  try {
    const res = await postfixApi.preview()
    preview.value = res.data
    showPreview.value = true
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.operationFailed'))
  }
}

function editConfig(row: PostfixConfig) {
  isEditing.value = true
  editingId.value = row.id
  form.value = { paramName: row.paramName, paramValue: row.paramValue, description: row.description || '' }
  showCreateDialog.value = true
}

async function saveConfig() {
  saving.value = true
  try {
    if (isEditing.value) {
      await postfixApi.updateConfig(editingId.value, { paramValue: form.value.paramValue })
    } else {
      await postfixApi.createConfig({ paramName: form.value.paramName, paramValue: form.value.paramValue, description: form.value.description })
    }
    ElMessage.success(t('common.saveSuccess'))
    showCreateDialog.value = false
    form.value = { paramName: '', paramValue: '', description: '' }
    isEditing.value = false
    editingId.value = ''
    await loadConfigs()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || t('common.operationFailed'))
  } finally {
    saving.value = false
  }
}

async function deleteConfig(row: PostfixConfig) {
  try {
    await ElMessageBox.confirm(t('common.deleteConfirm'), t('common.delete'), { type: 'warning' })
    await postfixApi.deleteConfig(row.id)
    ElMessage.success(t('common.deleteSuccess'))
    await loadConfigs()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || t('common.operationFailed'))
    }
  }
}
</script>

<style scoped>
.postfix-configs-page {
  height: 100%;
  margin: 0 auto;
}

.main-card {
  min-height: calc(100vh - 140px);
  border: none;
  box-shadow: var(--shadow-card);
}

.main-card :deep(.el-card__body) {
  padding: 12px 16px 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 10px;
}

.page-subtitle {
  font-size: 13px;
  color: var(--foreground-muted);
  margin: 2px 0 0 0;
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--foreground);
  margin: 0;
}

/* Compact top section */
.compact-top-section {
  margin-bottom: 12px;
}

.settings-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.settings-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.settings-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--foreground);
  white-space: nowrap;
}

.settings-select {
  width: 10rem !important;
  flex-shrink: 0;
}

.info-alert-inline {
  margin: 0;
  background-color: var(--surface-hover);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--warning-color);
  flex-shrink: 0;
}

.info-alert-inline :deep(.el-alert__content) {
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-alert-inline :deep(.el-alert__title) {
  font-size: 12px;
}

/* Variables hint */
.variables-hint {
  margin-top: 8px;
  padding: 8px 12px;
  background: var(--surface-hover);
  border-radius: 6px;
  border: 1px solid var(--border-color);
}

.hint-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--foreground-muted);
  margin-bottom: 6px;
}

.hint-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.var-tag {
  cursor: pointer;
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.var-tag:hover {
  opacity: 0.8;
}

.search-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background-color: var(--surface-hover);
  border-radius: 6px;
}

.search-bar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-bar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.config-preview {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  max-height: 60vh;
  overflow: auto;
  white-space: pre;
}

.preview-meta {
  font-size: 11px;
  color: var(--foreground-muted);
}

.pagination-container {
  margin-top: 12px;
  display: flex;
  justify-content: center;
}
</style>