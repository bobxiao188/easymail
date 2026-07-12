<template>
  <div class="classify-models-page">
    <el-card class="glass-card main-card">
      <el-tabs v-model="activeTab" class="model-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="FastText" name="FastText">
          <div class="tab-content">
            <div class="search-box">
              <div class="search-left">
                <el-select v-model="statusFilter" :placeholder="t('classifyModels.statusFilterPlaceholder')"
                  class="easy-select" style="width: 120px;" clearable @clear="loadModels" @change="loadModels">
                  <el-option :label="t('common.all')" value="" />
                  <el-option :label="t('classifyModels.enabled')" value="1" />
                  <el-option :label="t('classifyModels.disabled')" value="0" />
                </el-select>
                <el-input v-model="searchQuery" :placeholder="t('classifyModels.searchModel')" class="easy-input"
                  style="width: 260px;" clearable @clear="loadModels" @keyup.enter="loadModels" />
              </div>
              <div class="search-right">
                <el-button type="primary" size="default" @click="openAddModelDialog">
                  <el-icon :size="16"><Plus /></el-icon>
                  <span>{{ t('classifyModels.addModel') }}</span>
                </el-button>
                <el-button type="primary" size="default" @click="handleImportModels">
                  <el-icon :size="16"><Upload /></el-icon>
                  <span>{{ t('classifyModels.importModels') }}</span>
                </el-button>
              </div>
            </div>

            <div class="model-list">
              <el-table :data="models" class="easy-table">
                <el-table-column prop="id" :label="t('classifyModels.id')" width="80" />
                <el-table-column prop="name" :label="t('classifyModels.modelName')" />
                <el-table-column :label="t('classifyModels.classLabels')">
                  <template #default="scope">
                    <template v-if="scope.row.classLabels?.length">
                      <el-tag v-for="lb in scope.row.classLabels" :key="lb" size="small" class="language-tag">{{ lb }}</el-tag>
                    </template>
                    <span v-else>&mdash;</span>
                  </template>
                </el-table-column>
                <el-table-column prop="maxTextLength" :label="t('classifyModels.maxTextLength')" width="200" />
                <el-table-column prop="enabled" :label="t('classifyModels.status')" width="180">
                  <template #default="scope">
                    <el-tooltip :disabled="scope.row.activationReady || scope.row.enabled"
                      :content="t('classifyModels.activationBlockedHint')" placement="top">
                      <span class="activation-switch-wrap">
                        <el-switch v-model="scope.row.enabled"
                          :disabled="!scope.row.activationReady && !scope.row.enabled" class="easy-switch"
                          :active-value="true" :inactive-value="false" @change="handleStatusChange(scope.row)" />
                      </span>
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column prop="createdAt" :label="t('classifyModels.createTime')" width="200">
                  <template #default="scope">
                    <el-tooltip :content="formatDateTime(scope.row.createdAt)" placement="top">
                      {{ formatDate(scope.row.createdAt) }}
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column :label="t('classifyModels.operation')" width="180">
                  <template #default="scope">
                    <div class="action-buttons">
                      <el-button type="primary" size="small" class="easy-icon-button"
                        :title="t('classifyModels.tryPredict')" @click="handlePredictModel(scope.row)">
                        <el-icon :size="16"><Aim /></el-icon>
                      </el-button>
                      <el-button type="primary" size="small" class="easy-icon-button"
                        @click="openEditModelDialog(scope.row)">
                        <el-icon :size="16"><Edit /></el-icon>
                      </el-button>
                      <el-button type="danger" size="small" class="easy-icon-button"
                        @click="handleDeleteModel(scope.row.id)">
                        <el-icon :size="16"><Delete /></el-icon>
                      </el-button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination-container">
                <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[5, 10, 20]"
                  layout="total, sizes, prev, pager, next, jumper" :total="totalModels" @size-change="handleSizeChange"
                  @current-change="handleCurrentChange" class="easy-pagination" />
              </div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="DistilBERT" name="DistilBERT">
          <div class="tab-content">
            <div class="search-box">
              <div class="search-left">
                <el-select v-model="statusFilter" :placeholder="t('classifyModels.statusFilterPlaceholder')"
                  class="easy-select" style="width: 120px;" clearable @clear="loadModels" @change="loadModels">
                  <el-option :label="t('common.all')" value="" />
                  <el-option :label="t('classifyModels.enabled')" value="1" />
                  <el-option :label="t('classifyModels.disabled')" value="0" />
                </el-select>
                <el-input v-model="searchQuery" :placeholder="t('classifyModels.searchModel')" class="easy-input"
                  style="width: 260px;" clearable @clear="loadModels" @keyup.enter="loadModels" />
              </div>
              <div class="search-right">
                <el-button type="primary" size="default" @click="openAddModelDialog">
                  <el-icon :size="16"><Plus /></el-icon>
                  <span>{{ t('classifyModels.addModel') }}</span>
                </el-button>
                <el-button type="primary" size="default" @click="handleImportModels">
                  <el-icon :size="16"><Upload /></el-icon>
                  <span>{{ t('classifyModels.importModels') }}</span>
                </el-button>
              </div>
            </div>

            <div class="model-list">
              <el-table :data="models" class="easy-table">
                <el-table-column prop="id" :label="t('classifyModels.id')" width="80" />
                <el-table-column prop="name" :label="t('classifyModels.modelName')" />
                <el-table-column :label="t('classifyModels.classLabels')">
                  <template #default="scope">
                    <template v-if="scope.row.classLabels?.length">
                      <el-tag v-for="lb in scope.row.classLabels" :key="lb" size="small" class="language-tag">{{ lb }}</el-tag>
                    </template>
                    <span v-else>&mdash;</span>
                  </template>
                </el-table-column>
                <el-table-column prop="maxTextLength" :label="t('classifyModels.maxTextLength')" width="200" />
                <el-table-column prop="enabled" :label="t('classifyModels.status')" width="180">
                  <template #default="scope">
                    <el-tooltip :disabled="scope.row.activationReady || scope.row.enabled"
                      :content="t('classifyModels.activationBlockedHint')" placement="top">
                      <span class="activation-switch-wrap">
                        <el-switch v-model="scope.row.enabled"
                          :disabled="!scope.row.activationReady && !scope.row.enabled" class="easy-switch"
                          :active-value="true" :inactive-value="false" @change="handleStatusChange(scope.row)" />
                      </span>
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column prop="createdAt" :label="t('classifyModels.createTime')" width="200">
                  <template #default="scope">
                    <el-tooltip :content="formatDateTime(scope.row.createdAt)" placement="top">
                      {{ formatDate(scope.row.createdAt) }}
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column :label="t('classifyModels.operation')" width="210">
                  <template #default="scope">
                    <div class="action-buttons">
                      <el-button type="primary" size="small" class="easy-icon-button"
                        :title="t('classifyModels.tryPredict')" @click="handlePredictModel(scope.row)">
                        <el-icon :size="16"><Aim /></el-icon>
                      </el-button>
                      <el-button type="primary" size="small" class="easy-icon-button"
                        @click="openEditModelDialog(scope.row)">
                        <el-icon :size="16"><Edit /></el-icon>
                      </el-button>
                      <el-button type="danger" size="small" class="easy-icon-button"
                        @click="handleDeleteModel(scope.row.id)">
                        <el-icon :size="16"><Delete /></el-icon>
                      </el-button>
                      <el-button type="primary" size="small" class="easy-icon-button"
                        @click="handleExportModel(scope.row)">
                        <el-icon :size="16"><Download /></el-icon>
                      </el-button>
                    </div>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination-container">
                <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[5, 10, 20]"
                  layout="total, sizes, prev, pager, next, jumper" :total="totalModels" @size-change="handleSizeChange"
                  @current-change="handleCurrentChange" class="easy-pagination" />
              </div>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog :title="modelDialogTitle" v-model="modelDialogVisible" width="53%" class="easy-dialog" destroy-on-close
      align-center :close-on-click-modal="false" :close-on-press-escape="false">
      <el-form :model="modelForm" :rules="modelRules" ref="modelFormRef" label-width="160px" class="easy-form">
        <el-form-item :label="t('classifyModels.modelName')" prop="name">
          <el-input v-model="modelForm.name" :placeholder="namePlaceholder" class="easy-input" />
        </el-form-item>
        <el-form-item v-if="modelForm.id" :label="t('classifyModels.classLabels')">
          <div>
            <template v-if="modelForm.classLabels?.length">
              <el-tag v-for="lb in modelForm.classLabels" :key="lb" size="small" class="language-tag">{{ lb }}</el-tag>
            </template>
            <span v-else>&mdash;</span>
            <p class="form-hint">{{ t('classifyModels.classLabelsHint') }}</p>
          </div>
        </el-form-item>
        <el-form-item v-if="!isFastTextForm && !isDistilBERTForm" :label="t('classifyModels.algorithm')" prop="algorithm">
          <el-select v-model="modelForm.algorithm" :placeholder="t('classifyModels.selectAlgorithm')" class="easy-select">
            <el-option label="FastText" value="FastText" />
            <el-option label="DistilBERT" value="DistilBERT" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="!isFastTextForm && !isDistilBERTForm" :label="t('classifyModels.tokenizer')" prop="tokenizer">
          <el-select v-model="modelForm.tokenizer" :placeholder="t('classifyModels.selectTokenizer')" class="easy-select">
            <el-option :label="t('classifyModels.tokenizerWordPiece')" value="WordPiece" />
            <el-option :label="t('classifyModels.tokenizerDistilEnglish')" value="distilbert-base-cased" />
            <el-option :label="t('classifyModels.tokenizerDistilMultilingual')" value="distilbert-base-multilingual-cased" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="!isFastTextForm && !isDistilBERTForm" :label="t('classifyModels.supportedLanguages')" prop="languages">
          <el-checkbox-group v-model="modelForm.languages">
            <el-checkbox value="all">{{ t('classifyModels.allLanguages') }}</el-checkbox>
            <el-checkbox value="en">English</el-checkbox>
            <el-checkbox value="zh">{{ t('classifyModels.chinese') }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item v-if="!isDistilBERTForm" :label="t('classifyModels.modelParams')" prop="params">
          <el-input v-model="modelForm.params" :placeholder="t('classifyModels.modelParamsPlaceholder')"
            class="easy-input" type="textarea" :rows="5" />
        </el-form-item>
        <el-form-item v-if="isDistilBERTForm" :label="t('classifyModels.savePath')">
          <el-upload :key="onnxUploadKey" :auto-upload="false" :show-file-list="true" :limit="1" accept=".onnx"
            :on-change="handleDistilOnnxChange" :on-exceed="handleDistilOnnxExceed">
            <el-button type="primary">{{ t('classifyModels.onnxUpload') }}</el-button>
          </el-upload>
          <p class="form-hint">{{ t('classifyModels.onnxPickForSave') }}</p>
          <p v-if="modelForm.savePath" class="form-hint">
            {{ t('classifyModels.onnxCurrentPath') }}: {{ modelForm.savePath }}
          </p>
          <p class="form-hint">{{ t('classifyModels.onnxUploadHint') }}</p>
        </el-form-item>
        <el-form-item v-if="!isFastTextForm && !isDistilBERTForm" :label="t('classifyModels.savePath')" prop="savePath">
          <el-input v-model="modelForm.savePath" :placeholder="t('classifyModels.savePathPlaceholder')" class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('classifyModels.maxTextLength')" prop="maxTextLength">
          <el-input-number v-model="modelForm.maxTextLength" :min="isDistilBERTForm ? 10 : 1"
            :max="isDistilBERTForm ? 512 : 1000000" :step="10" class="easy-input-number" />
        </el-form-item>
        <el-form-item :label="t('classifyModels.emailFields')" prop="emailFields">
          <el-checkbox-group v-model="modelForm.emailFields">
            <el-checkbox value="from_name">{{ t('classifyModels.fromName') }}</el-checkbox>
            <el-checkbox value="subject">{{ t('classifyModels.subject') }}</el-checkbox>
            <el-checkbox value="html_body">{{ t('classifyModels.htmlBody') }}</el-checkbox>
            <el-checkbox value="plain_text_body">{{ t('classifyModels.plainTextBody') }}</el-checkbox>
            <el-checkbox value="attachment_names">{{ t('classifyModels.attachmentNames') }}</el-checkbox>
            <el-checkbox value="attach_body">{{ t('classifyModels.attachmentBody') }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button size="default" class="easy-button" @click="modelDialogVisible = false">
            <span>{{ t('common.cancel') }}</span>
          </el-button>
          <el-button type="primary" size="default" class="easy-button" @click="handleSaveModel">
            <span>{{ t('common.save') }}</span>
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="predictDialogVisible" :title="t('classifyModels.predictDialogTitle')" width="53%"
      class="easy-dialog" align-center destroy-on-close :close-on-click-modal="false">
      <p v-if="predictTarget" class="form-hint" style="margin-bottom: 8px">{{ predictTarget.name }}</p>
      <el-input v-model="predictForm.text" type="textarea" :rows="5"
        :placeholder="t('classifyModels.predictTextPlaceholder')" class="easy-input" />
      <div v-if="predictLangOptions.length" style="margin-top: 12px">
        <p class="form-hint">{{ t('classifyModels.predictLanguageHint') }}</p>
        <el-select v-model="predictForm.languageCodes" multiple clearable filterable class="easy-select"
          style="width: 100%; margin-top: 6px">
          <el-option v-for="opt in predictLangOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
      </div>
      <div v-if="predictResult" style="margin-top: 16px">
        <el-alert v-if="predictResult.predictError" type="error" :closable="false" :title="predictResult.predictError"
          show-icon />
        <template v-else>
          <p style="margin: 0 0 8px">
            <strong>{{ t('classifyModels.predictTopLabel') }}:</strong>
            {{ predictResult.topLabel }}
            <span class="form-hint"> ({{ formatPredictProb(predictResult.topProbability) }})</span>
          </p>
          <p class="form-hint" style="margin-bottom: 6px">{{ t('classifyModels.predictDistribution') }}</p>
          <el-table :data="predictResult.distribution" class="easy-table" size="small" max-height="240">
            <el-table-column prop="label" :label="t('classifyModels.sampleLabel')"/>
            <el-table-column :label="t('classifyModels.predictScore')">
              <template #default="scope">{{ formatPredictProb(scope.row.probability) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </div>
      <template #footer>
        <el-button class="easy-button" @click="predictDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" class="easy-button" :loading="predictLoading" @click="runPredict">
          {{ t('classifyModels.predictRun') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  classifyModelApi,
  type ClassifyModel,
  type ClassifyPredictResult,
  type OnnxModelParams
} from '../api/classifyModel'
import { ElMessage, ElMessageBox, type FormInstance, type UploadFile, type UploadFiles } from 'element-plus'
import { formatDate, formatDateTime } from '../utils/times'
import { Plus, Edit, Delete, Aim, Download, Upload } from '@element-plus/icons-vue'

const { t } = useI18n()

const models = ref<ClassifyModel[]>([])
const searchQuery = ref('')
const activeTab = ref('FastText')
const statusFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(5)
const totalModels = ref(0)

const modelDialogVisible = ref(false)
const modelDialogTitle = ref(t('classifyModels.addModel'))
const modelFormRef = ref<FormInstance | null>(null)

const isFastTextForm = computed(() => modelForm.algorithm === 'FastText')
const isDistilBERTForm = computed(() => modelForm.algorithm === 'DistilBERT')

const distilOnnxFile = ref<File | null>(null)
const onnxUploadKey = ref(0)

function handleDistilOnnxChange(_file: UploadFile, fileList: UploadFiles) {
  const raw = fileList[0]?.raw
  distilOnnxFile.value = raw ?? null
}

function handleDistilOnnxExceed(files: File[]) {
  distilOnnxFile.value = files[0] ?? null
}

function buildDistilBERTFormData(): FormData {
  const fd = new FormData()
  fd.append('name', modelForm.name.trim())
  fd.append('algorithm', 'DistilBERT')
  fd.append('tokenizer', 'distilbert-base-multilingual-cased')
  fd.append('languages', JSON.stringify(['all']))
  fd.append('emailFields', JSON.stringify(modelForm.emailFields))
  fd.append('maxTextLength', String(modelForm.maxTextLength))
  fd.append('params', JSON.stringify({ algorithm: 'DistilBERT', modelFile: 'model.onnx' }))
  if (distilOnnxFile.value) {
    fd.append('onnx', distilOnnxFile.value)
  }
  return fd
}

const namePlaceholder = computed(() =>
  isFastTextForm.value
    ? t('classifyModels.fasttextNamePlaceholder')
    : t('classifyModels.modelNamePlaceholder')
)

function formatModelLanguageTag(lang: string) {
  if (lang === 'en') return t('classifyModels.langEn')
  if (lang === 'zh') return t('classifyModels.langZh')
  if (lang === 'all') return t('classifyModels.allLanguages')
  return lang
}

const fullClassifyModelPayload = (model: ClassifyModel) => ({
  name: model.name,
  algorithm: model.algorithm,
  tokenizer: model.tokenizer,
  languages: model.languages,
  savePath: model.savePath ?? '',
  maxTextLength: model.maxTextLength,
  emailFields: model.emailFields,
  enabled: model.enabled,
  params: model.params
})

const modelForm = reactive({
  id: null as number | null,
  name: '',
  algorithm: 'DistilBERT',
  tokenizer: 'distilbert-base-cased',
  languages: ['all'],
  params: JSON.stringify({ algorithm: 'DistilBERT', modelFile: 'model.onnx' }, null, 2),
  savePath: '',
  maxTextLength: 256,
  emailFields: ['from_name', 'subject', 'plain_text_body', 'attachment_names'],
  enabled: false,
  classLabels: [] as string[]
})

const modelRules = computed(() => {
  const rules: Record<string, any[]> = {
    name: [
      { required: true, message: t('common.required'), trigger: 'blur' },
      { min: 2, max: 50, message: '', trigger: 'blur' }
    ],
    maxTextLength: [
      { required: true, message: '', trigger: 'blur' },
      { type: 'number', message: '', trigger: 'blur' }
    ],
    emailFields: [
      { required: true, message: '', trigger: 'blur' }
    ]
  }
  if (isFastTextForm.value) {
    rules.name.push({
      pattern: /^[a-zA-Z0-9_]+$/,
      message: t('classifyModels.fasttextNamePatternMsg'),
      trigger: 'blur'
    })
  } else if (isDistilBERTForm.value) {
    rules.maxTextLength.push({
      validator: (_rule: unknown, val: number, cb: (e?: Error) => void) => {
        if (typeof val !== 'number' || val < 10 || val > 512) {
          cb(new Error(t('classifyModels.maxTextLengthDistilbertRange')))
        } else {
          cb()
        }
      },
      trigger: 'blur'
    })
  } else {
    rules.algorithm = [{ required: true, message: '', trigger: 'blur' }]
    rules.tokenizer = [{ required: true, message: '', trigger: 'blur' }]
    rules.languages = [{ required: true, message: '', trigger: 'blur' }]
    rules.savePath = [{ required: true, message: '', trigger: 'blur' }]
  }
  return rules
})

const loadModels = async () => {
  try {
    const res = await classifyModelApi.list({
      keyword: searchQuery.value,
      algorithm: activeTab.value,
      status: statusFilter.value ? parseInt(statusFilter.value) : undefined,
      page: currentPage.value,
      pageSize: pageSize.value
    })
    if (res.code === 0) {
      models.value = res.data
      if (res.meta) {
        totalModels.value = (res.meta as any).total || 0
      }
    }
  } catch (error) {
    ElMessage.error(t('classifyModels.getModelListFailed'))
  }
}

const handleTabChange = () => {
  currentPage.value = 1
  loadModels()
}

const openAddModelDialog = () => {
  onnxUploadKey.value += 1
  distilOnnxFile.value = null
  modelDialogTitle.value = t('classifyModels.addModel')
  const isFastText = activeTab.value === 'FastText'
  Object.assign(modelForm, {
    id: null,
    name: '',
    algorithm: activeTab.value,
    tokenizer: isFastText ? 'gse' : 'distilbert-base-cased',
    languages: ['all'],
    params: isFastText
      ? JSON.stringify({
        algorithm: 'FastText',
        learningRate: 0.1,
        epoch: 5,
        wordNgrams: 2
      }, null, 2)
      : JSON.stringify({ algorithm: 'DistilBERT', modelFile: 'model.onnx' }, null, 2),
    savePath: '',
    maxTextLength: isFastText ? 4096 : 256,
    emailFields: ['from_name', 'subject', 'plain_text_body', 'attachment_names'],
    enabled: false,
    classLabels: [] as string[]
  })
  modelDialogVisible.value = true
}

const openEditModelDialog = (model: ClassifyModel) => {
  onnxUploadKey.value += 1
  distilOnnxFile.value = null
  modelDialogTitle.value = t('classifyModels.editModel')
  modelForm.id = model.id
  modelForm.name = model.name
  modelForm.algorithm = model.algorithm
  modelForm.tokenizer = model.tokenizer
  modelForm.languages = model.languages
  modelForm.params = typeof model.params === 'object' ? JSON.stringify(model.params, null, 2) : String(model.params ?? '')
  modelForm.savePath = model.savePath
  modelForm.maxTextLength =
    model.algorithm === 'DistilBERT'
      ? Math.min(512, Math.max(10, model.maxTextLength))
      : model.maxTextLength
  modelForm.emailFields = model.emailFields
  modelForm.enabled = model.enabled
  modelForm.classLabels = model.classLabels?.length ? [...model.classLabels] : []
  modelDialogVisible.value = true
}

const handleSaveModel = async () => {
  if (!modelFormRef.value) return

  try {
    const valid = await modelFormRef.value.validate()
    if (!valid) return

    const modelId = modelForm.id ? Number(modelForm.id) : null

    if (isDistilBERTForm.value) {
      if (!modelId && !distilOnnxFile.value) {
        ElMessage.warning(t('classifyModels.onnxRequiredOnCreate'))
        return
      }
      const fd = buildDistilBERTFormData()
      if (modelId !== null && modelId !== undefined && modelId > 0) {
        fd.append('enabled', modelForm.enabled ? 'true' : 'false')
        const res = await classifyModelApi.updateDistilBERT(modelId, fd)
        if (res.code !== 0) {
          ElMessage.error(res.message || t('classifyModels.operationFailed'))
          return
        }
        ElMessage.success(t('classifyModels.modelUpdateSuccess'))
      } else {
        const res = await classifyModelApi.createDistilBERT(fd)
        if (res.code !== 0) {
          ElMessage.error(res.message || t('classifyModels.operationFailed'))
          return
        }
        ElMessage.success(t('classifyModels.modelCreateSuccess'))
      }
      modelDialogVisible.value = false
      await loadModels()
      return
    }

    let paramsObj: OnnxModelParams
    try {
      paramsObj = JSON.parse(modelForm.params)
    } catch (error: any) {
      ElMessage.error(error.message || t('classifyModels.paramsFormatError'))
      return
    }

    const basePayload = {
      name: modelForm.name,
      algorithm: modelForm.algorithm,
      tokenizer: modelForm.tokenizer,
      languages: modelForm.languages,
      maxTextLength: modelForm.maxTextLength,
      emailFields: modelForm.emailFields,
      params: paramsObj,
      savePath: isFastTextForm.value ? (modelForm.savePath || '') : modelForm.savePath
    }

    if (modelId !== null && modelId !== undefined && modelId > 0) {
      await classifyModelApi.update(modelId, { ...basePayload, enabled: modelForm.enabled } as any)
      ElMessage.success(t('classifyModels.modelUpdateSuccess'))
    } else {
      await classifyModelApi.create(basePayload as any)
      ElMessage.success(t('classifyModels.modelCreateSuccess'))
    }
    modelDialogVisible.value = false
    await loadModels()
  } catch (error: any) {
    const errorMessage = error.response?.data?.message || error.message || t('classifyModels.operationFailed')
    ElMessage.error(errorMessage)
  }
}

const handleStatusChange = async (model: ClassifyModel) => {
  try {
    await classifyModelApi.update(model.id, { ...fullClassifyModelPayload(model), enabled: model.enabled })
    ElMessage.success(t('classifyModels.statusUpdateSuccess'))
  } catch (error: any) {
    model.enabled = !model.enabled
    const errorMessage = error.response?.data?.message || error.message || t('classifyModels.statusUpdateFailed')
    ElMessage.error(errorMessage)
  }
}

const handleDeleteModel = async (id: number) => {
  try {
    await ElMessageBox.confirm(
      t('classifyModels.deleteModelConfirm'),
      t('classifyModels.deleteConfirm'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )

    await classifyModelApi.delete(id)
    ElMessage.success(t('classifyModels.modelDeleteSuccess'))
    await loadModels()
  } catch (error: any) {
    if (error !== 'cancel') {
      const errorMessage = error.response?.data?.message || error.message || t('classifyModels.deleteFailed')
      ElMessage.error(errorMessage)
    }
  }
}

const importFileInput = ref<HTMLInputElement | null>(null)

function handleImportModels() {
  const algorithm = activeTab.value
  if (!importFileInput.value) {
    const el = document.createElement('input')
    el.type = 'file'
    el.accept = '.zip'
    el.style.display = 'none'
    el.addEventListener('change', async () => {
      const file = el.files?.[0]
      if (!file) return
      const fd = new FormData()
      fd.append('file', file)
      try {
        const res = await classifyModelApi.importModel(fd, algorithm)
        if (res.code !== 0) {
          ElMessage.error(res.message || t('classifyModels.operationFailed'))
          return
        }
        ElMessage.success(t('classifyModels.importModelSuccess'))
        await loadModels()
      } catch (e: any) {
        ElMessage.error(e.response?.data?.message || e.message || t('classifyModels.operationFailed'))
      } finally {
        el.value = ''
      }
    })
    document.body.appendChild(el)
    importFileInput.value = el
  }
  importFileInput.value.value = ''
  importFileInput.value.click()
}

async function handleExportModel(row: ClassifyModel) {
  try {
    await classifyModelApi.exportModel(row.id)
    ElMessage.success(t('classifyModels.exportModelSuccess'))
  } catch (e: any) {
    ElMessage.error(e.message || t('classifyModels.exportModelFailed'))
  }
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  loadModels()
}

const handleCurrentChange = (current: number) => {
  currentPage.value = current
  loadModels()
}

// ========== Predict Dialog ==========

const predictDialogVisible = ref(false)
const predictTarget = ref<ClassifyModel | null>(null)
const predictForm = reactive({ text: '', languageCodes: [] as string[] })
const predictResult = ref<ClassifyPredictResult | null>(null)
const predictLoading = ref(false)

const predictLangOptions = computed(() => {
  const m = predictTarget.value
  if (!m?.languages?.length) return [] as { value: string; label: string }[]
  return m.languages
    .filter((l) => l && l !== 'all')
    .map((l) => ({ value: l, label: formatModelLanguageTag(l) }))
})

function formatPredictProb(p: number) {
  if (typeof p !== 'number' || Number.isNaN(p)) return '&mdash;'
  return `${(p * 100).toFixed(2)}%`
}

function handlePredictModel(row: ClassifyModel) {
  predictTarget.value = row
  predictForm.text = ''
  predictForm.languageCodes = []
  predictResult.value = null
  predictDialogVisible.value = true
}

async function runPredict() {
  const m = predictTarget.value
  if (!m) return
  const text = predictForm.text.trim()
  if (!text) {
    ElMessage.warning(t('classifyModels.predictNeedText'))
    return
  }
  predictLoading.value = true
  predictResult.value = null
  try {
    const langs = predictForm.languageCodes.filter(Boolean)
    const res = await classifyModelApi.predict(m.id, {
      text,
      languageCodes: langs.length ? langs : undefined
    })
    if (res.code !== 0) {
      ElMessage.error(res.message || t('classifyModels.operationFailed'))
      return
    }
    predictResult.value = res.data
  } catch (e: any) {
    ElMessage.error(e.response?.data?.message || e.message || t('classifyModels.operationFailed'))
  } finally {
    predictLoading.value = false
  }
}

onMounted(async () => {
  await loadModels()
})
</script>

<style scoped>
.classify-models-page {
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

.model-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.model-tabs :deep(.el-tabs__header) {
  margin: 0 0 16px;
  padding: 0;
  border-bottom: 1px solid var(--border-default);
}

.model-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
}

.model-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow-y: auto;
}

.tab-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding-top: 20px;
}

.search-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 15px;
  margin-bottom: 20px;
  background: var(--surface-light);
  border-radius: 8px;
}

.search-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.model-list {
  flex: 1;
  overflow-y: auto;
}

.pagination-container {
  display: flex;
  justify-content: center;
  padding: 15px 0;
}

.dialog-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.language-tag {
  margin-right: 4px;
}

.form-hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--foreground-muted);
  line-height: 1.5;
}

.activation-switch-wrap {
  display: inline-block;
  vertical-align: middle;
}

.easy-icon-button {
  padding: 6px;
}
</style>