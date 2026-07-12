<template>
  <div class="scanner-page">
    <el-card class="glass-card main-card">
      <el-tabs v-model="activeTab" class="scanner-tabs">
        <el-tab-pane :label="t('FilterRules.builtinFeatures')" name="features">
          <div class="toolbar">
            <el-button type="primary" class="easy-button" @click="loadFeatures">
              <el-icon>
                <Refresh />
              </el-icon>
              {{ t('common.refresh') }}
            </el-button>
          </div>
          <el-table v-loading="featuresLoading" :data="features" class="easy-table" stripe>
            <el-table-column prop="featureKey" :label="t('FilterRules.featureKey')" min-width="160" />
            <el-table-column prop="label" :label="t('FilterRules.name')" min-width="120" />
            <el-table-column prop="valueType" :label="t('FilterRules.type')" width="160" />
            <el-table-column prop="unit" :label="t('FilterRules.unit')" width="120" />
            <el-table-column prop="description" :label="t('FilterRules.description')" min-width="200" show-overflow-tooltip />
          </el-table>
          <div class="pagination-container">
            <el-pagination class="easy-pagination" v-model:current-page="featurePage"
              v-model:page-size="featurePageSize" :page-sizes="[10, 20, 50, 100]" :small="small" :disabled="disabled"
              :background="background" layout="total, sizes, prev, pager, next, jumper" :total="featureTotal"
              @size-change="handleFeatureSizeChange" @current-change="handleFeatureCurrentChange" />
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('FilterRules.customFeatures')" name="custom">
          <div class="toolbar">
            <el-button type="primary" class="easy-button" @click="openCustomDialog()">
              <el-icon>
                <Plus />
              </el-icon>
              {{ t('FilterRules.addFeature') }}
            </el-button>
            <el-button class="easy-button" @click="loadCustomFeatures">
              <el-icon>
                <Refresh />
              </el-icon>
              {{ t('common.refresh') }}
            </el-button>
          </div>
          <el-table v-loading="customLoading" :data="customFeatures" class="easy-table" stripe>
            <el-table-column prop="featureKey" :label="t('FilterRules.featureKey')" min-width="160" />
            <el-table-column prop="label" :label="t('FilterRules.name')" min-width="160" />
            <el-table-column prop="type" :label="t('FilterRules.type')" />
            <el-table-column prop="valueType" :label="t('FilterRules.valueType')" width="160" />
            <el-table-column prop="enabled" :label="t('FilterRules.enabled')" width="160">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" :active-value="true" :inactive-value="false" active-text=""
                  inactive-text="" @change="handleCustomEnabledChange(row)" />
              </template>
            </el-table-column>
            <el-table-column prop="updatedAt" :label="t('FilterRules.updateTime')" width="220">
              <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
            </el-table-column>
            <el-table-column :label="t('common.action')" width="160">
              <template #default="{ row }">
                <div class="action-buttons">
                  <el-button type="primary" class="easy-icon-button" size="small" @click="openCustomDialog(row)">
                    <el-icon>
                      <Edit />
                    </el-icon>
                  </el-button>
                  <el-button type="danger" class="easy-icon-button" size="small" @click="handleDeleteCustom(row)">
                    <el-icon>
                      <Delete />
                    </el-icon>
                  </el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <div class="pagination-container">
            <el-pagination class="easy-pagination" v-model:current-page="customPage" v-model:page-size="customPageSize"
              :page-sizes="[10, 20, 50, 100]" :small="small" :disabled="disabled" :background="background"
              layout="total, sizes, prev, pager, next, jumper" :total="customTotal"
              @size-change="handleCustomSizeChange" @current-change="handleCustomCurrentChange" />
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('FilterRules.rules')" name="rules">
          <div class="toolbar">
            <el-button type="primary" class="easy-button" @click="openRuleDialog()">
              <el-icon>
                <Plus />
              </el-icon>
              {{ t('FilterRules.addRule') }}
            </el-button>
            <el-button class="easy-button" @click="loadRules">
              <el-icon>
                <Refresh />
              </el-icon>
              {{ t('common.refresh') }}
            </el-button>
          </div>
          <el-table v-loading="rulesLoading" :data="rules" class="easy-table" stripe>
            <el-table-column prop="id" :label="t('common.id')" width="80" />
            <el-table-column prop="name" :label="t('FilterRules.name')" />
            <el-table-column :label="t('FilterRules.conditionJson')" min-width="200">
              <template #default="{ row }">
                <span class="mono-ellipsis" :title="row.conditionJson">{{ truncate(row.conditionJson, 80) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="enabled" :label="t('FilterRules.enabled')" width="160">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" :active-value="true" :inactive-value="false" active-text=""
                  inactive-text="" @change="handleRuleEnabledChange(row)" />
              </template>
            </el-table-column>
            <el-table-column prop="priority" :label="t('FilterRules.priority')" width="160" sortable />
            <el-table-column prop="action" :label="t('FilterRules.action')" width="160">
              <template #default="{ row }">
                <el-tag size="small" :type="actionTagType(row.action)">{{ row.action }}</el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="updatedAt" :label="t('FilterRules.updateTime')" width="220">
              <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
            </el-table-column>
            <el-table-column :label="t('common.action')" width="160">
              <template #default="{ row }">
                <div class="action-buttons">
                  <el-button type="primary" class="easy-icon-button" size="small" @click="openRuleDialog(row)">
                    <el-icon>
                      <Edit />
                    </el-icon>
                  </el-button>
                  <el-button type="danger" class="easy-icon-button" size="small" @click="handleDeleteRule(row)">
                    <el-icon>
                      <Delete />
                    </el-icon>
                  </el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <div class="pagination-container">
            <el-pagination class="easy-pagination" v-model:current-page="rulePage" v-model:page-size="rulePageSize"
              :page-sizes="[10, 20, 50, 100]" :small="small" :disabled="disabled" :background="background"
              layout="total, sizes, prev, pager, next, jumper" :total="ruleTotal" @size-change="handleRuleSizeChange"
              @current-change="handleRuleCurrentChange" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog v-model="ruleDialogVisible" :title="ruleForm.id ? t('FilterRules.editRule') : t('FilterRules.addRule')" width="920px"
      class="easy-dialog rule-dialog" destroy-on-close align-center :close-on-click-modal="false"
      :close-on-press-escape="false" @opened="onRuleDialogOpened">
      <el-form ref="ruleFormRef" :model="ruleForm" :rules="ruleFormRules" label-width="150px" class="easy-form">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('FilterRules.name')" prop="name">
              <el-input v-model="ruleForm.name" :placeholder="t('FilterRules.ruleNamePlaceholder')" class="easy-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('FilterRules.enabled')" prop="enabled">
              <el-switch v-model="ruleForm.enabled" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('FilterRules.priority')" prop="priority">
              <el-input-number v-model="ruleForm.priority" :step="1" class="easy-input-number" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('FilterRules.action')" prop="action">
              <el-select v-model="ruleForm.action" class="easy-select" style="width: 100%">
                <el-option :label="t('FilterRules.actionAccept')" value="accept" />
                <el-option :label="t('FilterRules.actionSpam')" value="spam" />
                <el-option :label="t('FilterRules.actionQuarantine')" value="quarantine" />
                <el-option :label="t('FilterRules.actionReject')" value="reject" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="24">
            <el-form-item :label="t('FilterRules.condition')" class="cond-form-item" required>
              <div class="cond-builder-wrap">
                <ConditionNodeEditor v-model="conditionRoot" :features="featuresAll" :depth="0" />
                <el-collapse v-model="advCollapse" class="json-collapse">
                  <el-collapse-item :title="t('FilterRules.advanced')" name="json">
                    <pre class="json-pre" tabindex="0">{{ jsonFormatted }}</pre>
                    <div class="json-actions">
                      <el-button class="easy-button" size="small" @click="copyJson">{{ t('FilterRules.copyJson') }}</el-button>
                      <el-button class="easy-button" type="primary" size="small" @click="openJsonEditDialog">{{ t('FilterRules.manualEditJson') }}</el-button>
                    </div>
                  </el-collapse-item>
                </el-collapse>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="ruleDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button class="easy-button" type="primary" :loading="ruleSaving" @click="submitRule">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog class="easy-dialog json-edit-dialog" v-model="jsonEditVisible" :title="t('FilterRules.editJson')" width="640px"
      destroy-on-close align-center :close-on-click-modal="false" :close-on-press-escape="false"
      @opened="onJsonEditOpened">
      <el-input v-model="jsonEditDraft" type="textarea" :rows="16" class="mono-input json-edit-ta"
        :placeholder="t('FilterRules.conditionJsonPlaceholder')" />
      <template #footer>
        <el-button class="easy-button" @click="jsonEditVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button class="easy-button" type="primary" @click="applyJsonDraft">{{ t('FilterRules.applyToVisual') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="customDialogVisible" :title="customForm.id ? t('FilterRules.editCustomFeature') : t('FilterRules.addCustomFeature')" width="920px"
      class="easy-dialog rule-dialog" destroy-on-close align-center :close-on-click-modal="false"
      :close-on-press-escape="false" @opened="onCustomDialogOpened">
      <el-form ref="customFormRef" :model="customForm" :rules="customFormRules" label-width="150px" class="easy-form">
        <el-form-item :label="t('FilterRules.featureKey')" prop="featureKey">
          <el-input v-model="customForm.featureKey" :placeholder="t('FilterRules.featureKeyPlaceholder')" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('FilterRules.name')" prop="label">
          <el-input v-model="customForm.label" :placeholder="t('FilterRules.featureNamePlaceholder')" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('FilterRules.enabled')" prop="enabled">
          <el-switch v-model="customForm.enabled" />
        </el-form-item>
        <el-form-item :label="t('FilterRules.type')" prop="type">
          <el-select v-model="customForm.type" class="easy-select" style="width: 100%">
            <el-option :label="t('FilterRules.typeMetaRegex')" value="meta_regex" />
            <el-option :label="t('FilterRules.typeComposite')" value="composite" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('FilterRules.valueType')" prop="valueType">
          <el-select v-model="customForm.valueType" class="easy-select" style="width: 100%">
            <el-option :label="t('FilterRules.valueTypeBool')" value="bool" />
            <el-option :label="t('FilterRules.valueTypeNumber')" value="number" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('FilterRules.description')">
          <el-input v-model="customForm.description" :placeholder="t('FilterRules.optional')" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('FilterRules.unit')">
          <el-input v-model="customForm.unit" :placeholder="t('FilterRules.optional')" class="easy-input" />
        </el-form-item>

        <template v-if="customForm.type === 'meta_regex'">
          <el-form-item :label="t('FilterRules.matchFields')" required>
            <el-select v-model="metaSources" class="easy-select" multiple searchable style="width: 100%"
              :placeholder="t('FilterRules.selectMetaFields')">
              <el-option :label="t('FilterRules.fieldConnectIp')" value="connect_ip" />
              <el-option :label="t('FilterRules.fieldMailFrom')" value="mail_from" />
              <el-option :label="t('FilterRules.fieldRcpt')" value="rcpt" />
              <el-option :label="t('FilterRules.fieldSubject')" value="subject" />
              <el-option :label="t('FilterRules.fieldHeaderFromEmail')" value="header_from_email" />
              <el-option :label="t('FilterRules.fieldHeaderFromName')" value="header_from_name" />
              <el-option :label="t('FilterRules.fieldBody')" value="body" />
              <el-option :label="t('FilterRules.fieldAttachmentNames')" value="attachment_names" />
              <el-option :label="t('FilterRules.fieldUrlList')" value="url_list" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('FilterRules.regexPattern')" required>
            <el-input v-model="metaPattern" :placeholder="t('FilterRules.regexPlaceholder')" class="easy-input" />
          </el-form-item>
          <el-form-item :label="t('FilterRules.flags')">
            <el-checkbox-group v-model="metaFlags">
              <el-checkbox value="i">{{ t('FilterRules.flagI') }}</el-checkbox>
              <el-checkbox value="m">{{ t('FilterRules.flagM') }}</el-checkbox>
              <el-checkbox value="s">{{ t('FilterRules.flagS') }}</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item :label="t('FilterRules.mode')">
            <el-select v-model="metaMode" class="easy-select" style="width: 100%">
              <el-option :label="t('FilterRules.modeAny')" value="any" />
              <el-option :label="t('FilterRules.modeAll')" value="all" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('FilterRules.emit')">
            <el-select v-model="metaEmit" class="easy-select" style="width: 100%">
              <el-option :label="t('FilterRules.emitBoolHit')" value="bool_hit" />
              <el-option :label="t('FilterRules.emitCount')" value="count" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('FilterRules.specJson')">
            <el-input v-model="customForm.specJson" type="textarea" :rows="6" class="mono-input json-edit-ta" />
          </el-form-item>
        </template>

        <template v-else-if="customForm.type === 'composite'">
          <el-form-item :label="t('FilterRules.compositeCondition')" required>
            <div class="cond-builder-wrap">
              <ConditionNodeEditor v-model="customConditionRoot" :features="featuresAll" :depth="0" />
              <el-collapse v-model="customAdvCollapse" class="json-collapse">
                <el-collapse-item :title="t('FilterRules.advanced')" name="json">
                  <pre class="json-pre" tabindex="0">{{ customJsonFormatted }}</pre>
                  <div class="json-actions">
                    <el-button class="easy-button" size="small" @click="copyCustomJson">{{ t('FilterRules.copyJson') }}</el-button>
                    <el-button class="easy-button" type="primary" size="small" @click="openCustomJsonEditDialog">{{ t('FilterRules.manualEditJson') }}</el-button>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </el-form-item>
          <el-form-item :label="t('FilterRules.specJson')">
            <el-input v-model="customForm.specJson" type="textarea" :rows="6" class="mono-input json-edit-ta" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="customDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button class="easy-button" type="primary" :loading="customSaving" @click="submitCustom">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="customJsonEditVisible" :title="t('FilterRules.editJson')" width="640px" destroy-on-close align-center
      :close-on-click-modal="false" :close-on-press-escape="false" @opened="onCustomJsonEditOpened">
      <el-input v-model="customJsonEditDraft" type="textarea" :rows="16" class="mono-input json-edit-ta"
        :placeholder="t('FilterRules.conditionJsonPlaceholder')" />
      <template #footer>
        <el-button class="easy-button" @click="customJsonEditVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button class="easy-button" type="primary" @click="applyCustomJsonDraft">{{ t('FilterRules.applyToVisual') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { scannerApi, type FilterFeature, type FilterRule, type FilterRuleAction, type ScannerCustomFeatureDef } from '../api/filter.ts'
import { formatDateTime } from '../utils/times'
import ConditionNodeEditor from '../components/filter/ConditionNodeEditor.vue'
import type { CondNode } from '../components/filter/condTypes.ts'
import {
  defaultRoot,
  parseConditionJson,
  stringifyCondition,
  validateConditionTree
} from '../components/filter/condTypes.ts'

const { t } = useI18n()

const activeTab = ref<'features' | 'custom' | 'rules'>('features')
const features = ref<FilterFeature[]>([]) // table (paged)
const featuresAll = ref<FilterFeature[]>([]) // for condition editor (full list)
const rules = ref<FilterRule[]>([])
const featuresLoading = ref(false)
const rulesLoading = ref(false)
const customLoading = ref(false)

const featurePage = ref(1)
const featurePageSize = ref(10)
const featureTotal = ref(0)

const rulePage = ref(1)
const rulePageSize = ref(10)
const ruleTotal = ref(0)

const customFeatures = ref<ScannerCustomFeatureDef[]>([])
const customPage = ref(1)
const customPageSize = ref(10)
const customTotal = ref(0)

const small = ref(false)
const disabled = ref(false)
const background = ref(true)

const ruleDialogVisible = ref(false)
const ruleSaving = ref(false)
const ruleFormRef = ref<FormInstance | null>(null)

const conditionRoot = ref<CondNode>(defaultRoot())
const advCollapse = ref<string[]>([])

const jsonEditVisible = ref(false)
const jsonEditDraft = ref('')

const customDialogVisible = ref(false)
const customSaving = ref(false)
const customFormRef = ref<FormInstance | null>(null)
const customConditionRoot = ref<CondNode>(defaultRoot())
const customAdvCollapse = ref<string[]>([])
const customJsonEditVisible = ref(false)
const customJsonEditDraft = ref('')

const metaSources = ref<string[]>([])
const metaPattern = ref('')
const metaFlags = ref<string[]>([])
const metaMode = ref<'any' | 'all'>('any')
const metaEmit = ref<'bool_hit' | 'count'>('bool_hit')

const customForm = reactive({
  id: null as number | null,
  featureKey: '',
  label: '',
  type: 'meta_regex' as 'meta_regex' | 'composite',
  valueType: 'bool' as 'bool' | 'number',
  enabled: true,
  specJson: '',
  description: '',
  unit: ''
})

const customJsonFormatted = computed(() => stringifyCondition(customConditionRoot.value, true))

const ruleForm = reactive({
  id: null as number | null,
  name: '',
  enabled: true,
  priority: 0,
  action: 'accept' as FilterRuleAction,
  conditionJson: ''
})

const jsonFormatted = computed(() => stringifyCondition(conditionRoot.value, true))

watch(
  conditionRoot,
  (v) => {
    ruleForm.conditionJson = stringifyCondition(v, false)
  },
  { deep: true }
)

const ruleFormRules: FormRules = {
  name: [{ required: true, message: t('common.required', { field: t('FilterRules.name') }), trigger: 'blur' }],
  action: [{ required: true, message: t('common.selectRequired', { field: t('FilterRules.action') }), trigger: 'change' }]
}

const customFormRules: FormRules = {
  featureKey: [{ required: true, message: t('common.required', { field: t('FilterRules.featureKey') }), trigger: 'blur' }],
  label: [{ required: true, message: t('common.required', { field: t('FilterRules.name') }), trigger: 'blur' }],
  type: [{ required: true, message: t('common.selectRequired', { field: t('FilterRules.type') }), trigger: 'change' }],
  valueType: [{ required: true, message: t('common.selectRequired', { field: t('FilterRules.valueType') }), trigger: 'change' }]
}

function truncate(s: string, n: number) {
  if (!s) return ''
  return s.length <= n ? s : s.slice(0, n) + '…'
}

function actionTagType(a: string) {
  if (a === 'spam') return 'warning'
  if (a === 'quarantine') return 'danger'
  if (a === 'reject') return 'danger'
  return 'success'
}

async function loadFeatures() {
  featuresLoading.value = true
  try {
    const res = await scannerApi.listFeatures({ page: featurePage.value, pageSize: featurePageSize.value })
    if (res.code === 0 && Array.isArray(res.data)) {
      features.value = res.data
      featureTotal.value = (res as any).meta?.total ?? res.data.length
    } else {
      ElMessage.error(res.message || t('FilterRules.loadFeaturesFailed'))
    }
  } catch {
    ElMessage.error(t('FilterRules.loadFeaturesFailed'))
  } finally {
    featuresLoading.value = false
  }
}

async function loadFeaturesAll() {
  // For rule builder dropdown: always load full merged feature list (built-in + enabled custom).
  try {
    const res = await scannerApi.listFeatures()
    if (res.code === 0 && Array.isArray(res.data)) {
      featuresAll.value = res.data
    }
  } catch {
    // ignore best-effort
  }
}

async function loadRules() {
  rulesLoading.value = true
  try {
    const res = await scannerApi.listRules({ page: rulePage.value, pageSize: rulePageSize.value })
    if (res.code === 0 && Array.isArray(res.data)) {
      rules.value = res.data
      ruleTotal.value = (res as any).meta?.total ?? res.data.length
    } else {
      ElMessage.error(res.message || t('FilterRules.loadRulesFailed'))
    }
  } catch {
    ElMessage.error(t('FilterRules.loadRulesFailed'))
  } finally {
    rulesLoading.value = false
  }
}

async function loadCustomFeatures() {
  customLoading.value = true
  try {
    const res = await scannerApi.listCustomFeatures({ page: customPage.value, pageSize: customPageSize.value })
    if (res.code === 0 && Array.isArray(res.data)) {
      customFeatures.value = res.data
      customTotal.value = (res as any).meta?.total ?? res.data.length
    } else {
      ElMessage.error(res.message || t('FilterRules.loadCustomFeaturesFailed'))
    }
  } catch {
    ElMessage.error(t('FilterRules.loadCustomFeaturesFailed'))
  } finally {
    customLoading.value = false
  }
}

function handleFeatureSizeChange(size: number) {
  featurePageSize.value = size
  featurePage.value = 1
  void loadFeatures()
}

function handleFeatureCurrentChange(page: number) {
  featurePage.value = page
  void loadFeatures()
}

function handleRuleSizeChange(size: number) {
  rulePageSize.value = size
  rulePage.value = 1
  void loadRules()
}

function handleRuleCurrentChange(page: number) {
  rulePage.value = page
  void loadRules()
}

function handleCustomSizeChange(size: number) {
  customPageSize.value = size
  customPage.value = 1
  void loadCustomFeatures()
}

function handleCustomCurrentChange(page: number) {
  customPage.value = page
  void loadCustomFeatures()
}

function openRuleDialog(row?: FilterRule) {
  void loadFeaturesAll()
  if (row) {
    ruleForm.id = row.id
    ruleForm.name = row.name
    ruleForm.enabled = row.enabled
    ruleForm.priority = row.priority
    ruleForm.action = (row.action as FilterRuleAction) || 'accept'
    conditionRoot.value = parseConditionJson(row.conditionJson || '')
  } else {
    ruleForm.id = null
    ruleForm.name = ''
    ruleForm.enabled = true
    ruleForm.priority = 0
    ruleForm.action = 'accept'
    conditionRoot.value = defaultRoot()
  }
  ruleForm.conditionJson = stringifyCondition(conditionRoot.value, false)
  advCollapse.value = []
  ruleDialogVisible.value = true
}

function onRuleDialogOpened() {
  ruleFormRef.value?.clearValidate()
}

function onCustomDialogOpened() {
  customFormRef.value?.clearValidate()
}

function buildMetaSpecJson() {
  const spec = {
    sources: metaSources.value,
    pattern: metaPattern.value,
    flags: metaFlags.value.join(''),
    mode: metaMode.value,
    emit: metaEmit.value
  }
  customForm.specJson = JSON.stringify(spec, null, 2)
}

function buildCompositeSpecJson() {
  const spec = {
    conditionJson: stringifyCondition(customConditionRoot.value, false),
    emit: 'bool'
  }
  customForm.specJson = JSON.stringify(spec, null, 2)
}

watch([metaSources, metaPattern, metaFlags, metaMode, metaEmit], () => {
  if (customForm.type === 'meta_regex') buildMetaSpecJson()
})

watch(
  customConditionRoot,
  () => {
    if (customForm.type === 'composite') buildCompositeSpecJson()
  },
  { deep: true }
)

function openCustomDialog(row?: ScannerCustomFeatureDef) {
  if (row) {
    customForm.id = row.id
    customForm.featureKey = row.featureKey
    customForm.label = row.label
    customForm.type = row.type
    customForm.valueType = row.valueType
    customForm.enabled = row.enabled
    customForm.specJson = row.specJson || ''
    customForm.description = row.description || ''
    customForm.unit = row.unit || ''

    if (row.type === 'meta_regex') {
      try {
        const s = JSON.parse(row.specJson || '{}')
        metaSources.value = Array.isArray(s.sources) ? s.sources : []
        metaPattern.value = String(s.pattern || '')
        metaFlags.value = String(s.flags || '').split('').filter((x: string) => ['i', 'm', 's'].includes(x))
        metaMode.value = s.mode === 'all' ? 'all' : 'any'
        metaEmit.value = s.emit === 'count' ? 'count' : 'bool_hit'
      } catch {
        metaSources.value = []
        metaPattern.value = ''
        metaFlags.value = []
        metaMode.value = 'any'
        metaEmit.value = 'bool_hit'
      }
      buildMetaSpecJson()
    } else {
      try {
        const s = JSON.parse(row.specJson || '{}')
        customConditionRoot.value = parseConditionJson(String(s.conditionJson || ''))
      } catch {
        customConditionRoot.value = defaultRoot()
      }
      buildCompositeSpecJson()
    }
  } else {
    customForm.id = null
    customForm.featureKey = ''
    customForm.label = ''
    customForm.type = 'meta_regex'
    customForm.valueType = 'bool'
    customForm.enabled = true
    customForm.description = ''
    customForm.unit = ''
    metaSources.value = []
    metaPattern.value = ''
    metaFlags.value = []
    metaMode.value = 'any'
    metaEmit.value = 'bool_hit'
    customConditionRoot.value = defaultRoot()
    buildMetaSpecJson()
  }
  customAdvCollapse.value = []
  customDialogVisible.value = true
}

async function submitCustom() {
  if (!customFormRef.value) return
  await customFormRef.value.validate(async (valid) => {
    if (!valid) return
    // Ensure specJson is synced
    if (customForm.type === 'meta_regex') buildMetaSpecJson()
    if (customForm.type === 'composite') buildCompositeSpecJson()

    customSaving.value = true
    try {
      const payload = {
        featureKey: customForm.featureKey.trim(),
        label: customForm.label.trim(),
        type: customForm.type,
        valueType: customForm.valueType,
        enabled: customForm.enabled,
        specJson: customForm.specJson,
        description: customForm.description?.trim() || '',
        unit: customForm.unit?.trim() || ''
      }
      if (customForm.id != null) {
        const res = await scannerApi.updateCustomFeature(customForm.id, payload)
        if (res.code === 0) {
          ElMessage.success(t('FilterRules.saveSuccess'))
          customDialogVisible.value = false
          await loadCustomFeatures()
          await loadFeatures()
        } else {
          ElMessage.error(res.message || t('FilterRules.saveFailed'))
        }
      } else {
        const res = await scannerApi.createCustomFeature(payload)
        if (res.code === 0) {
          ElMessage.success(t('FilterRules.createSuccess'))
          customDialogVisible.value = false
          await loadCustomFeatures()
          await loadFeatures()
        } else {
          ElMessage.error(res.message || t('FilterRules.createFailed'))
        }
      }
    } catch (e: unknown) {
      const msg = e && typeof e === 'object' && 'message' in e ? String((e as Error).message) : t('FilterRules.requestFailed')
      ElMessage.error(msg)
    } finally {
      customSaving.value = false
    }
  })
}

async function handleDeleteCustom(row: ScannerCustomFeatureDef) {
  try {
    await ElMessageBox.confirm(`${t('FilterRules.deleteConfirm')}${row.label}`, t('common.confirm'), { type: 'warning' })
    const res = await scannerApi.deleteCustomFeature(row.id)
    if (res.code === 0) {
      ElMessage.success(t('FilterRules.deleteSuccess'))
      if (customFeatures.value.length <= 1 && customPage.value > 1) {
        customPage.value -= 1
      }
      await loadCustomFeatures()
      await loadFeatures()
    } else {
      ElMessage.error(res.message || t('FilterRules.deleteFailed'))
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('FilterRules.deleteFailed'))
  }
}

async function copyCustomJson() {
  try {
    await navigator.clipboard.writeText(customJsonFormatted.value)
    ElMessage.success(t('FilterRules.copySuccess'))
  } catch {
    ElMessage.warning(t('FilterRules.copyFailed'))
  }
}

function openCustomJsonEditDialog() {
  customJsonEditDraft.value = customJsonFormatted.value
  customJsonEditVisible.value = true
}

function onCustomJsonEditOpened() {
  customJsonEditDraft.value = customJsonFormatted.value
}

function applyCustomJsonDraft() {
  const next = parseConditionJson(customJsonEditDraft.value)
  const err = validateConditionTree(next)
  if (err) {
    ElMessage.error(err)
    return
  }
  customConditionRoot.value = next
  customJsonEditVisible.value = false
  ElMessage.success(t('FilterRules.applyToVisualSuccess'))
}

function openJsonEditDialog() {
  jsonEditDraft.value = jsonFormatted.value
  jsonEditVisible.value = true
}

function onJsonEditOpened() {
  jsonEditDraft.value = jsonFormatted.value
}

function applyJsonDraft() {
  const next = parseConditionJson(jsonEditDraft.value)
  const err = validateConditionTree(next)
  if (err) {
    ElMessage.error(err)
    return
  }
  conditionRoot.value = next
  jsonEditVisible.value = false
  ElMessage.success(t('FilterRules.applyToVisualSuccess'))
}

async function copyJson() {
  try {
    await navigator.clipboard.writeText(jsonFormatted.value)
    ElMessage.success(t('FilterRules.copySuccess'))
  } catch {
    ElMessage.warning(t('FilterRules.copyFailed'))
  }
}

async function submitRule() {
  if (!ruleFormRef.value) return
  const err = validateConditionTree(conditionRoot.value)
  if (err) {
    ElMessage.error(err)
    return
  }
  ruleForm.conditionJson = stringifyCondition(conditionRoot.value, false)

  await ruleFormRef.value.validate(async (valid) => {
    if (!valid) return
    ruleSaving.value = true
    try {
      const payload = {
        name: ruleForm.name.trim(),
        enabled: ruleForm.enabled,
        priority: ruleForm.priority,
        action: ruleForm.action,
        conditionJson: ruleForm.conditionJson.trim()
      }
      if (ruleForm.id != null) {
        const res = await scannerApi.updateRule(ruleForm.id, payload)
        if (res.code === 0) {
          ElMessage.success(t('FilterRules.saveSuccess'))
          ruleDialogVisible.value = false
          await loadRules()
        } else {
          ElMessage.error(res.message || t('FilterRules.saveFailed'))
        }
      } else {
        const res = await scannerApi.createRule(payload)
        if (res.code === 0) {
          ElMessage.success(t('FilterRules.createSuccess'))
          ruleDialogVisible.value = false
          await loadRules()
        } else {
          ElMessage.error(res.message || t('FilterRules.createFailed'))
        }
      }
    } catch (e: unknown) {
      const msg = e && typeof e === 'object' && 'message' in e ? String((e as Error).message) : t('FilterRules.requestFailed')
      ElMessage.error(msg)
    } finally {
      ruleSaving.value = false
    }
  })
}

async function handleDeleteRule(row: FilterRule) {
  try {
    await ElMessageBox.confirm(`${t('FilterRules.deleteConfirm')}${t('FilterRules.rule.name')}`, t('common.confirm'), { type: 'warning' })
    const res = await scannerApi.deleteRule(row.id)
    if (res.code === 0) {
      ElMessage.success(t('FilterRules.deleteSuccess'))
      // If we deleted the last item on a page, step back one page (best-effort).
      if (rules.value.length <= 1 && rulePage.value > 1) {
        rulePage.value -= 1
      }
      await loadRules()
    } else {
      ElMessage.error(res.message || t('FilterRules.deleteFailed'))
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('FilterRules.deleteFailed'))
  }
}

// 处理自定义特征启用状态改变
async function handleCustomEnabledChange(row: ScannerCustomFeatureDef) {
  try {
    const res = await scannerApi.patchCustomFeature(row.id, { enabled: row.enabled })
    if (res.code === 0) {
      ElMessage.success(t('FilterRules.updateSuccess'))
      await loadCustomFeatures()
    } else {
      ElMessage.error(res.message || t('FilterRules.updateFailed'))
    }
  } catch (e) {
    ElMessage.error(t('FilterRules.updateFailed'))
  }
}

// 处理规则启用状态改变
async function handleRuleEnabledChange(row: FilterRule) {
  try {
    const res = await scannerApi.patchRule(row.id, { enabled: row.enabled })
    if (res.code === 0) {
      ElMessage.success(t('FilterRules.updateSuccess'))
      await loadRules()
    } else {
      ElMessage.error(res.message || t('FilterRules.updateFailed'))
    }
  } catch (e) {
    ElMessage.error(t('FilterRules.updateFailed'))
  }
}

onMounted(() => {
  void loadFeatures()
  void loadFeaturesAll()
  void loadCustomFeatures()
  void loadRules()
})
</script>

<style scoped>
.scanner-page {
  height: 100%;
  margin: 0 auto;
}

.main-card {
  min-height: calc(100vh - 140px);
  border: none;
  box-shadow: var(--shadow-card);
}

.main-card :deep(.el-card__header) {
  background: transparent;
  border-bottom: 1px solid var(--border-default);
  padding: 20px 25px;
}

.main-card :deep(.el-card__body) {
  padding: 20px 25px 28px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
}

.toolbar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.hint {
  font-size: 13px;
  color: var(--foreground-muted);
  line-height: 1.5;
}

.hint code {
  font-size: 0.8rem;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--border-default);
}

.mono-ellipsis {
  font-size: 0.8rem;
}

.rule-dialog :deep(.el-dialog__body) {
  max-height: min(78vh, 860px);
  overflow-y: auto;
}

.cond-form-item :deep(.el-form-item__content) {
  display: block;
}

.cond-builder-wrap {
  width: 100%;
}

.cond-intro {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.json-collapse {
  margin-top: 16px;
  border: none;
  --el-collapse-header-bg-color: transparent;
  --el-collapse-content-bg-color: transparent;
}

.json-collapse :deep(.el-collapse-item__header) {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  background: transparent !important;
}

.json-collapse :deep(.el-collapse-item__wrap) {
  background: transparent !important;
  border-bottom: none;
}

.json-collapse :deep(.el-collapse-item__content) {
  background: transparent !important;
  padding-bottom: 0;
}

.json-pre {
  margin: 0 0 12px;
  padding: 12px;
  border-radius: 8px;
  background: var(--surface-light);
  border: 1px solid var(--border-default);
  color: var(--foreground);
  line-height: 1.6;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 240px;
  overflow-y: auto;
}

.json-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.json-edit-ta :deep(textarea) {
  background: var(--surface-light) !important;
  color: var(--foreground) !important;
  border-color: var(--border-default) !important;
}

.scanner-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}
</style>
