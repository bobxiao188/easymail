<template>
  <div class="sample-form-dialogs">
    <el-button link size="default" @click="openAddSample()">
      {{ t('training.addSample') }}
    </el-button>

    <!-- Add Sample Dialog -->
    <el-dialog v-model="addSampleVisible" :title="t('training.addSample')" width="53%"
      class="easy-dialog" align-center :close-on-click-modal="false">
      <el-form ref="addSampleFormRef" :model="addSampleForm" :rules="addSampleRules"
        label-width="150px" class="easy-form">
        <el-form-item :label="t('training.category')" prop="categoryId">
          <el-select v-model="addSampleForm.categoryId" :placeholder="t('training.category')"
            class="easy-select" filterable>
            <el-option v-for="c in categoryList" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('training.tag')" prop="tag">
          <el-select v-model="addSampleForm.tag" :placeholder="t('training.tag')"
            class="easy-select" filterable allow-create clearable>
            <el-option v-for="tag in tagOptions" :key="tag" :label="tag" :value="tag" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('training.sampleText')" prop="text">
          <el-input v-model="addSampleForm.text" type="textarea" :rows="5"
            :placeholder="t('training.sampleText')" class="easy-input" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="addSampleVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" class="easy-button" @click="handleSaveAddSample">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Edit Sample Dialog -->
    <el-dialog v-model="editSampleVisible" :title="t('training.editSample')" width="53%"
      class="easy-dialog" align-center :close-on-click-modal="false">
      <el-form ref="editSampleFormRef" :model="editSampleForm" :rules="editSampleRules"
        label-width="150px" class="easy-form">
        <el-form-item :label="t('training.category')" prop="categoryId">
          <el-select v-model="editSampleForm.categoryId" :placeholder="t('training.category')"
            class="easy-select" filterable>
            <el-option v-for="c in categoryList" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('training.tag')" prop="tag">
          <el-select v-model="editSampleForm.tag" :placeholder="t('training.tag')"
            class="easy-select" filterable allow-create clearable>
            <el-option v-for="tag in editTagOptions" :key="tag" :label="tag" :value="tag" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('training.sampleText')" prop="text">
          <el-input v-model="editSampleForm.text" type="textarea" :rows="5"
            :placeholder="t('training.sampleText')" class="easy-input" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="editSampleVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" class="easy-button" @click="handleSaveEditSample">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  trainingApi,
  type Sample,
  type SampleCategory
} from '../../api/training'

interface Props {
  categoryList: SampleCategory[]
  tagsMap: Record<number, string[]>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'sample-updated': []
}>()

const { t } = useI18n()

// ========== Add Sample ==========
const addSampleVisible = ref(false)
const addSampleFormRef = ref()
const addSampleForm = reactive({
  categoryId: undefined as number | undefined,
  tag: '',
  text: ''
})

const addSampleRules = {
  categoryId: [
    { required: true, message: () => t('training.categoryNameRequired') || 'Please select a category', trigger: 'change' }
  ],
  tag: [
    { required: true, message: () => t('common.required', { field: t('training.tag') }) || 'Please enter tag', trigger: 'blur' }
  ],
  text: [
    { required: true, message: () => t('common.required', { field: t('training.sampleText') }) || 'Please enter sample text', trigger: 'blur' }
  ]
}

// Tag options for the selected category (add dialog)
const tagOptions = computed(() => {
  const cid = addSampleForm.categoryId
  return cid ? (props.tagsMap[cid] || []) : []
})

// Watch: when add dialog category changes, update tag options
watch(() => addSampleForm.categoryId, () => {
  addSampleForm.tag = ''
})

function openAddSample() {
  addSampleForm.categoryId = undefined
  addSampleForm.tag = ''
  addSampleForm.text = ''
  addSampleFormRef.value?.clearValidate()
  addSampleVisible.value = true
}

async function handleSaveAddSample() {
  if (!addSampleFormRef.value) return

  try {
    await addSampleFormRef.value.validate()
  } catch {
    return
  }

  const categoryId = addSampleForm.categoryId!
  const tag = addSampleForm.tag.trim()
  const text = addSampleForm.text.trim()

  try {
    const res = await trainingApi.createSample({ categoryId, tag, text })
    if (res.code !== 0) {
      ElMessage.error(res.message || t('common.operationFailed'))
      return
    }
    ElMessage.success(t('training.sampleAdded'))
    addSampleForm.categoryId = undefined
    addSampleForm.tag = ''
    addSampleForm.text = ''
    addSampleVisible.value = false
    emit('sample-updated')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
  }
}

// ========== Edit Sample ==========
const editSampleVisible = ref(false)
const editSampleFormRef = ref()
const editSampleForm = reactive({
  id: 0,
  categoryId: undefined as number | undefined,
  tag: '',
  text: ''
})

const editSampleRules = {
  categoryId: [
    { required: true, message: () => t('training.categoryNameRequired') || 'Please select a category', trigger: 'change' }
  ],
  tag: [
    { required: true, message: () => t('common.required', { field: t('training.tag') }) || 'Please enter tag', trigger: 'blur' }
  ],
  text: [
    { required: true, message: () => t('common.required', { field: t('training.sampleText') }) || 'Please enter sample text', trigger: 'blur' }
  ]
}

// Tag options for the selected category (edit dialog)
const editTagOptions = computed(() => {
  const cid = editSampleForm.categoryId
  return cid ? (props.tagsMap[cid] || []) : []
})

// Watch: when edit dialog category changes, update tag options
watch(() => editSampleForm.categoryId, () => {
  editSampleForm.tag = ''
})

function openEditSample(sample: Sample) {
  editSampleForm.id = sample.id
  editSampleForm.categoryId = sample.categoryId
  editSampleForm.tag = sample.tag
  editSampleForm.text = sample.text
  editSampleFormRef.value?.clearValidate()
  editSampleVisible.value = true
}

async function handleSaveEditSample() {
  if (!editSampleFormRef.value) return

  try {
    await editSampleFormRef.value.validate()
  } catch {
    return
  }

  const categoryId = editSampleForm.categoryId!
  const tag = editSampleForm.tag.trim()
  const text = editSampleForm.text.trim()

  try {
    const res = await trainingApi.updateSample(editSampleForm.id, { categoryId, tag, text })
    if (res.code !== 0) {
      ElMessage.error(res.message || t('common.operationFailed'))
      return
    }
    ElMessage.success(t('training.sampleUpdated'))
    editSampleForm.id = 0
    editSampleForm.categoryId = undefined
    editSampleForm.tag = ''
    editSampleForm.text = ''
    editSampleVisible.value = false
    emit('sample-updated')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
  }
}

// Expose methods to parent
defineExpose({
  openAddSample,
  openEditSample
})
</script>

<style scoped>
.easy-button {
  --el-button-bg-color: var(--surface-card);
  --el-button-text-color: var(--foreground);
  --el-button-border-color: var(--border-default);
  --el-button-hover-bg-color: var(--surface-light);
  --el-button-hover-text-color: var(--accent);
  --el-button-active-bg-color: var(--accent);
  --el-button-active-text-color: var(--surface-card);
}

.easy-button.is-active {
  --el-button-bg-color: var(--accent);
  --el-button-text-color: var(--surface-card);
}

.easy-button:hover:not(.is-active) {
  --el-button-bg-color: var(--surface-light);
}

.easy-button:active:not(.is-active) {
  --el-button-bg-color: var(--accent);
  --el-button-text-color: var(--surface-card);
}

.easy-button:disabled {
  --el-button-bg-color: var(--surface-light);
  --el-button-text-color: var(--foreground-muted);
  opacity: 0.6;
  cursor: not-allowed;
}

.easy-dialog :deep(.el-dialog__body) {
  padding: 20px;
}

.easy-form :deep(.el-form-item__label) {
  color: var(--foreground);
  font-weight: 500;
}

.easy-input :deep(.el-input__wrapper) {
  background: var(--surface-card);
  border-color: var(--border-default);
  color: var(--foreground);
}

.easy-input :deep(.el-input__wrapper:hover) {
  border-color: var(--accent);
}

.easy-input :deep(.el-input__wrapper.is-focus) {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--accent) inset;
}

.easy-input :deep(.el-input__inner) {
  color: var(--foreground);
}

.easy-select :deep(.el-input__wrapper) {
  background: var(--surface-card);
  border-color: var(--border-default);
  color: var(--foreground);
}

.easy-select :deep(.el-input__wrapper:hover) {
  border-color: var(--accent);
}

.easy-select :deep(.el-input__wrapper.is-focus) {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--accent) inset;
}

.easy-select :deep(.el-input__inner) {
  color: var(--foreground);
}

.easy-select :deep(.el-select__dropdown) {
  background: var(--surface-card);
  border-color: var(--border-default);
}

.easy-select :deep(.el-select__wrapper.is-focus) {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--accent) inset;
}
</style>