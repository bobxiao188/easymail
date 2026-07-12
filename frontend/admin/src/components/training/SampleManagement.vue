<template>
  <div class="sample-management">
    <div class="toolbar">
      <div class="toolbar-left">
        <el-select v-model="sampleFilter.categoryId" clearable :placeholder="t('training.category')"
          class="easy-select" style="width: 150px;" @change="onSampleFilterChange" filterable>
          <el-option v-for="c in categoryList" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
        <el-select v-model="sampleFilter.tag" clearable :placeholder="t('training.tag')"
          class="easy-select" style="width: 150px;" @change="onSampleFilterChange">
          <el-option v-for="t in tagOptions" :key="t" :label="t" :value="t" />
        </el-select>
        <el-input v-model="sampleFilter.keyword" :placeholder="t('training.searchSamples')"
          class="easy-input" style="width: 220px;" clearable @clear="loadSamples"
          @keyup.enter="loadSamples" />
        <el-button class="easy-button" @click="loadSamples">{{ t('common.search') }}</el-button>
      </div>
      <div class="toolbar-right">
        <el-button class="easy-button" :disabled="sampleRows.length === 0" @click="toggleSelectAll">
          {{ selectedIds.length > 0 ? t('training.deselectAll') : t('training.selectAll') }}
        </el-button>
        <el-dropdown trigger="click" :disabled="selectedIds.length === 0" @command="onBatchCommand">
          <el-button class="easy-button" type="warning">
            {{ t('training.batchOperations') }}
            <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="batchDelete">{{ t('training.batchDelete') }}</el-dropdown-item>
              <el-dropdown-item command="batchTransfer">{{ t('training.batchTransfer') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <CategoryDialog
          :category-list="categoryList"
          @category-updated="emit('category-updated')"
        />
        <SampleForm
          ref="sampleFormRef"
          :category-list="categoryList"
          :tags-map="props.tagsMap"
          @sample-updated="emit('sample-updated')"
        />
        <BatchImportDialog
          :category-list="categoryList"
          @samples-updated="emit('sample-updated')"
        />
        <SampleStatsDialog :category-id="sampleFilter.categoryId" />
      </div>
    </div>

    <!-- Sample table -->
    <el-table :data="sampleRows" class="easy-table samples-table" stripe @selection-change="onSelectionChange" ref="sampleTableRef">
      <el-table-column type="selection" width="55" />
      <el-table-column prop="id" :label="t('common.id')" width="120" />
      <el-table-column prop="category" :label="t('training.category')" width="200" show-overflow-tooltip />
      <el-table-column prop="tag" :label="t('training.tag')" width="150" show-overflow-tooltip />
      <el-table-column :label="t('training.sampleText')" min-width="200" show-overflow-tooltip>
        <template #default="scope">
          <el-tooltip placement="bottom" effect="dark" :show-after="200"
            :popper-options="{ strategy: 'fixed' }" :z-index="4100"
            popper-class="samples-text-tooltip-popper" teleported>
            <template #content>
              <div class="samples-text-tooltip-body">{{ scope.row.text }}</div>
            </template>
            <span class="samples-text-cell">{{ scope.row.text }}</span>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.operation')" width="160">
        <template #default="scope">
          <el-button type="primary" link size="small" @click="sampleFormRef?.openEditSample(scope.row)">
            <el-icon :size="16"><Edit /></el-icon>
          </el-button>
          <el-button type="danger" link size="small" @click="handleDeleteSample(scope.row)">
            <el-icon :size="16"><Delete /></el-icon>
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-container">
      <el-pagination v-model:current-page="samplePage" v-model:page-size="samplePageSize"
        :page-sizes="[10, 15, 20, 50]" layout="total, sizes, prev, pager, next"
        :total="sampleTotal" @size-change="handleSampleSizeChange"
        @current-change="handleSamplePageChange" class="easy-pagination" />
    </div>

    <!-- Batch Transfer Dialog -->
    <el-dialog v-model="transferDialogVisible" :title="t('training.batchTransferTitle')" width="520px" destroy-on-close :modal="true" class="easy-dialog">
      <el-form :model="transferForm" label-width="180px">
        <el-form-item :label="t('training.selectTargetCategory')">
          <el-select v-model="transferForm.categoryId" :placeholder="t('training.selectTargetCategory')"
            class="easy-select" style="width: 100%;" filterable @change="onTransferCategoryChange">
            <el-option v-for="c in categoryList" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('training.selectTargetTag')">
          <el-select v-model="transferForm.tag" :placeholder="t('training.selectTargetTag')"
            class="easy-select" style="width: 100%;" filterable>
            <el-option v-for="tag in transferTagOptions" :key="tag" :label="tag" :value="tag" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="transferDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button class="easy-button" type="primary" :disabled="!transferForm.categoryId || !transferForm.tag"
          @click="confirmBatchTransfer">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Delete, ArrowDown } from '@element-plus/icons-vue'
import {
  trainingApi,
  type Sample,
  type SampleCategory
} from '@/api/training'
import CategoryDialog from './CategoryDialog.vue'
import BatchImportDialog from './BatchImportDialog.vue'
import SampleForm from './SampleForm.vue'
import SampleStatsDialog from './SampleStatsDialog.vue'

interface Props {
  categoryList: SampleCategory[]
  tagsMap: Record<number, string[]>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'sample-updated': []
  'category-updated': []
  'tags-loaded': [categoryId: number, tags: string[]]
}>()

const { t } = useI18n()

const sampleFormRef = ref()
const sampleTableRef = ref()

// ========== Selection State ==========
const selectedIds = ref<number[]>([])

function onSelectionChange(rows: Sample[]) {
  selectedIds.value = rows.map(r => r.id)
}

function toggleSelectAll() {
  if (selectedIds.value.length > 0) {
    // Deselect all
    sampleTableRef.value?.clearSelection()
    selectedIds.value = []
  } else {
    // Select all
    sampleTableRef.value?.toggleAllSelection()
  }
}

// ========== Batch Transfer State ==========
const transferDialogVisible = ref(false)
const transferForm = reactive({
  categoryId: undefined as number | undefined,
  tag: ''
})

const transferTagOptions = computed(() => {
  const cid = transferForm.categoryId
  return cid ? (props.tagsMap[cid] || []) : []
})

async function onTransferCategoryChange() {
  transferForm.tag = ''
  if (!transferForm.categoryId) return
  try {
    const res = await trainingApi.listTags(transferForm.categoryId)
    if (res.code === 0 && Array.isArray(res.data)) {
      emit('tags-loaded', transferForm.categoryId, res.data)
    }
  } catch {
    // ignore
  }
}

// ========== Batch Operations ==========
function onBatchCommand(command: string) {
  if (command === 'batchDelete') {
    handleBatchDelete()
  } else if (command === 'batchTransfer') {
    openTransferDialog()
  }
}

function openTransferDialog() {
  transferForm.categoryId = undefined
  transferForm.tag = ''
  transferDialogVisible.value = true
}

async function confirmBatchTransfer() {
  if (!transferForm.categoryId || !transferForm.tag) {
    ElMessage.warning(t('training.selectTargetCategory'))
    return
  }
  try {
    await trainingApi.batchUpdateSamples(selectedIds.value, transferForm.categoryId, transferForm.tag)
    ElMessage.success(t('training.batchTransferred', { count: selectedIds.value.length }))
    transferDialogVisible.value = false
    await loadSamples()
    emit('sample-updated')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
  }
}

async function handleBatchDelete() {
  try {
    await ElMessageBox.confirm(
      t('training.batchDeleteConfirm', { count: selectedIds.value.length }),
      t('common.confirm'),
      { confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel'), type: 'warning' }
    )
    await trainingApi.batchDeleteSamples(selectedIds.value)
    ElMessage.success(t('training.batchDeleted', { count: selectedIds.value.length }))
    await loadSamples()
    emit('sample-updated')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
    }
  }
}

// ========== Sample Management ==========

// Filter state
const sampleFilter = reactive({
  categoryId: undefined as number | undefined,
  tag: '',
  keyword: ''
})

// Tag options for the selected category
const tagOptions = computed(() => {
  const cid = sampleFilter.categoryId
  return cid ? (props.tagsMap[cid] || []) : []
})

// Data state
const sampleRows = ref<Sample[]>([])
const samplePage = ref(1)
const samplePageSize = ref(10)
const sampleTotal = ref(0)

// Auto-load tags when categoryId changes
watch(() => sampleFilter.categoryId, async (newId) => {
  sampleFilter.tag = ''
  if (!newId) return
  loadTags()
})

async function loadTags() {
  if (!sampleFilter.categoryId) {
    return
  }
  try {
    const res = await trainingApi.listTags(sampleFilter.categoryId)
    if (res.code === 0 && Array.isArray(res.data)) {
      emit('tags-loaded', sampleFilter.categoryId, res.data)
    }
  } catch {
    // ignore
  }
}

async function loadSamples() {
  try {
    const res = await trainingApi.listSamples({
      categoryId: sampleFilter.categoryId || undefined,
      tag: sampleFilter.tag || undefined,
      keyword: sampleFilter.keyword || undefined,
      page: samplePage.value,
      pageSize: samplePageSize.value
    })
    if (res.code === 0) {
      sampleRows.value = res.data || []
      sampleTotal.value = (res.meta as any)?.total ?? 0
      // Clear selection after loading
      selectedIds.value = []
      sampleTableRef.value?.clearSelection()
    }
  } catch {
    ElMessage.error(t('common.loadFailed'))
  }
}

function onSampleFilterChange() {
  samplePage.value = 1
  loadSamples()
}

async function handleDeleteSample(row: Sample) {
  try {
    await ElMessageBox.confirm(
      t('training.deleteSampleConfirm'),
      t('common.confirm'),
      { confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel'), type: 'warning' }
    )
    await trainingApi.deleteSample(row.id)
    ElMessage.success(t('training.sampleDeleted'))
    await loadSamples()
    emit('sample-updated')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
    }
  }
}

const handleSampleSizeChange = (size: number) => {
  samplePageSize.value = size
  samplePage.value = 1
  loadSamples()
}

const handleSamplePageChange = (p: number) => {
  samplePage.value = p
  loadSamples()
}

onMounted(() => {
  loadSamples()
})
</script>

<style scoped lang="scss">
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 15px;
  margin-bottom: 20px;
  background: var(--surface-light);
  border-radius: 8px;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.samples-table {
  flex: 1;
  overflow-y: auto;
}

.samples-text-cell {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}

.pagination-container {
  display: flex;
  justify-content: center;
  padding: 15px 0;
}

// Responsive
@media (max-width: 1400px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  .toolbar-left,
  .toolbar-right {
    justify-content: space-between;
  }
}
</style>

<style>
.samples-text-tooltip-popper {
  max-width: min(480px, 88vw) !important;
  box-sizing: border-box;
}

.samples-text-tooltip-popper .samples-text-tooltip-body {
  max-width: 100%;
  white-space: normal;
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.55;
  text-align: left;
}

.pagination-container :deep(.el-pagination) {
  --el-pagination-bg-color: var(--surface-card);
}

.pagination-container :deep(.el-pagination .el-pagination__total) {
  color: var(--foreground-muted);
}

.pagination-container :deep(.el-pagination .el-pager li) {
  color: var(--foreground);
  background: var(--surface-light);
}

.pagination-container :deep(.el-pagination .el-pager li.is-active) {
  background: var(--accent);
  color: var(--surface-card);
}
</style>