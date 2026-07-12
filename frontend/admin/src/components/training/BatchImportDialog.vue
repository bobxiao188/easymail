<template>
  <div class="batch-import-dialog">
    <el-button link size="default" @click="openBatchImport">
      {{ t('training.batchImport') }}
    </el-button>

    <!-- Batch Import Dialog -->
    <el-dialog v-model="showBatchImport" :title="t('training.batchImportTitle')" width="53%"
      class="easy-dialog" align-center destroy-on-close :close-on-click-modal="false">
      <div class="batch-import-toolbar">
        <p class="form-hint" style="margin-bottom:12px">{{ t('training.batchImportHint') }}</p>
        <el-button type="primary" link size="small" @click="addTagFileRow">
          <el-icon :size="16">
            <Plus />
          </el-icon>
        </el-button>
      </div>
      <el-table :data="tagFileUploadRows" size="small" class="easy-table">
        <el-table-column :label="t('training.category')" min-width="160">
          <template #default="{ row }">
            <el-select v-model="row.categoryId" filterable :placeholder="t('training.category')"
              :class="{ 'is-error': tagFileRowErrors[row.id]?.categoryId }"
              @change="(cid: number) => { loadTagsByCategoryId(cid); clearTagFileRowError(row.id, 'categoryId'); }">
              <el-option v-for="c in categoryList" :key="c.id" :label="c.name" :value="c.id" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column :label="t('training.tag')" min-width="160">
          <template #default="{ row }">
            <el-autocomplete
              v-model="row.tag"
              :fetch-suggestions="makeTagSuggestions(row)"
              :placeholder="t('training.tag')"
              :class="{ 'is-error': tagFileRowErrors[row.id]?.tag }"
              class="tag-autocomplete"
              @change="clearTagFileRowError(row.id, 'tag')"
            />
          </template>
        </el-table-column>
        <el-table-column :label="t('training.batchImportColFile')" min-width="200">
          <template #default="{ row }">
            <el-upload :show-file-list="false" accept=".txt,text/plain"
              :auto-upload="false" :disabled="samplesUploading"
              :on-change="bindTagFileUpload(row)">
              <el-button size="small" type="primary" link>
                {{ row.file ? row.file.name : t('training.batchImportPickFile') }}
              </el-button>
            </el-upload>
          </template>
        </el-table-column>
        <el-table-column :label="t('training.batchImportColAction')" width="120" align="center">
          <template #default="{ row }">
            <el-button type="danger" link size="small" @click="removeTagFileRow(row.id)">
              <el-icon :size="14">
                <Delete />
              </el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button class="easy-button" @click="closeBatchImport">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" class="easy-button" :loading="samplesUploading"
          @click="handleBatchImport">{{ t('training.batchImportImport') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
import {
  trainingApi,
  type SampleCategory
} from '@/api/training'

interface Props {
  categoryList: SampleCategory[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'samples-updated': []
}>()

const { t } = useI18n()

const showBatchImport = ref(false)
const samplesUploading = ref(false)

interface TagFileRow {
  id: number
  categoryId: number | undefined
  tag: string
  tags: string[]
  file: File | null
}
let tagFileRowIdSeq = 0
function makeTagFileRow(): TagFileRow {
  tagFileRowIdSeq += 1
  return { id: tagFileRowIdSeq, categoryId: undefined, tag: '', tags: [], file: null }
}

const tagFileUploadRows = ref<TagFileRow[]>([makeTagFileRow()])
const tagFileRowErrors = ref<Record<number, { categoryId?: true; tag?: true }>>({})

// Single tag cache shared across rows
const tagCache = ref<Record<number, string[]>>({})

function loadTagsByCategoryId(cid: number) {
  if (!cid || tagCache.value[cid]) return
  trainingApi.listTags(cid).then(res => {
    if (res.code === 0 && Array.isArray(res.data)) {
      tagCache.value[cid] = res.data.sort((a: string, b: string) => a.localeCompare(b))
    }
  }).catch(() => {})
}

function makeTagSuggestions(row: TagFileRow) {
  return (query: string, cb: (items: { value: string }[]) => void) => {
    const cid = row.categoryId
    if (!cid) { cb([]); return }
    const allTags = tagCache.value[cid] || []
    const filtered = query ? allTags.filter((t: string) => t.toLowerCase().includes(query.toLowerCase())) : allTags
    cb(filtered.map((t: string) => ({ value: t })))
  }
}

function clearTagFileRowError(id: number, field: 'categoryId' | 'tag') {
  const err = tagFileRowErrors.value[id]
  if (!err) return
  if (field === 'categoryId') delete err.categoryId
  else delete err.tag
  if (!Object.keys(err).length) {
    delete tagFileRowErrors.value[id]
  }
}

function addTagFileRow() {
  tagFileUploadRows.value.push(makeTagFileRow())
}

function removeTagFileRow(id: number) {
  if (tagFileUploadRows.value.length <= 1) {
    const r = tagFileUploadRows.value[0]
    if (r) { r.categoryId = undefined; r.tag = ''; r.tags = []; r.file = null }
    return
  }
  tagFileUploadRows.value = tagFileUploadRows.value.filter((x: TagFileRow) => x.id !== id)
}

function bindTagFileUpload(row: TagFileRow) {
  return (uf: any) => { row.file = uf.raw ?? null }
}

const SAMPLE_UPLOAD_BATCH = 200

interface LineToSampleTextResult {
  categoryId: number
  tag: string
  text: string
}

function lineToSampleText(line: string, categoryId: number, tag: string): LineToSampleTextResult | null {
  const trimmed = line.trim()
  if (!trimmed) return null
  return { categoryId, tag, text: trimmed }
}

async function handleBatchImport() {
  // Clear previous errors
  tagFileRowErrors.value = {}
  const rowsWithFile = tagFileUploadRows.value.filter((r: TagFileRow) => r.file)
  if (!rowsWithFile.length) {
    ElMessage.warning(t('training.batchImportNoFile'))
    return
  }
  let hasError = false
  for (const r of rowsWithFile) {
    const errs: { categoryId?: true; tag?: true } = {}
    if (!r.categoryId) errs.categoryId = true
    if (!r.tag.trim()) errs.tag = true
    if (Object.keys(errs).length) {
      tagFileRowErrors.value[r.id] = errs
      hasError = true
    }
  }
  if (hasError) {
    ElMessage.warning(t('training.batchImportNeedTag'))
    return
  }
  samplesUploading.value = true
  try {
    const items: LineToSampleTextResult[] = []
    for (const r of rowsWithFile) {
      const categoryId = r.categoryId!
      const tag = r.tag.trim()
      const text = await (r.file as File).text()
      for (const line of text.split(/\r?\n/)) {
        const it = lineToSampleText(line, categoryId, tag)
        if (it) items.push(it)
      }
    }
    if (!items.length) {
      ElMessage.warning(t('training.sampleUploadNoValidLines'))
      return
    }
    for (let i = 0; i < items.length; i += SAMPLE_UPLOAD_BATCH) {
      const chunk = items.slice(i, i + SAMPLE_UPLOAD_BATCH)
      const res = await trainingApi.createSamplesBatch({ items: chunk })
      if (res.code !== 0) {
        ElMessage.error(res.message || t('training.sampleUploadFailed'))
        return
      }
    }
    ElMessage.success(t('training.sampleUploadSuccess', { n: items.length }))
    showBatchImport.value = false
    emit('samples-updated')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.message || e.message || t('training.sampleUploadFailed'))
  } finally {
    samplesUploading.value = false
  }
}

function openBatchImport() {
  tagFileUploadRows.value = [makeTagFileRow()]
  showBatchImport.value = true
}

function closeBatchImport() {
  tagFileUploadRows.value = [makeTagFileRow()]
  showBatchImport.value = false
}
</script>

<style scoped>
.batch-import-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 15px;
  margin-bottom: 20px;
  background: var(--surface-light);
  border-radius: 8px;
}

.form-hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--foreground-muted);
  line-height: 1.5;
}

.tag-autocomplete {
  background: var(--surface-light) !important;
  border: 1px solid var(--border-default) !important;
  border-radius: 4px !important;
  padding: 0 8px !important;
  height: 32px !important;
  font-size: 14px !important;
  color: var(--foreground) !important;
  font-weight: 500 !important;
  line-height: 1.5 !important;
}

.tag-autocomplete .el-input__wrapper {
  background: transparent !important;
  box-shadow: none !important;
}

.tag-autocomplete .el-input__inner {
  background: transparent !important;
}

.tag-autocomplete .el-input__wrapper.is-focus {
  border-color: var(--el-color-primary) !important;
  box-shadow: 0 0 0 1px var(--el-color-primary) inset !important;
}

.tag-autocomplete .el-input__inner.is-focus {
  border-color: var(--el-color-primary) !important;
}

.el-select.is-error .el-select__wrapper {
  box-shadow: 0 0 0 1px var(--el-color-danger) inset !important;
}

.tag-autocomplete.is-error {
  border-color: var(--el-color-danger) !important;
}

.easy-dialog :deep(.el-dialog__body) {
  padding: 20px;
}
</style>