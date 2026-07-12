<template>
  <div class="category-dialog">
    <el-button link size="default" @click="showCategoryDialog = true">
      {{ t('training.manageCategories') }}
    </el-button>

    <!-- Category Management Dialog -->
    <el-dialog v-model="showCategoryDialog" :title="t('training.manageCategories')" width="53%"
      class="easy-dialog" destroy-on-close align-center :close-on-click-modal="false">
      <div class="category-management-toolbar">
        <div class="toolbar-left">
          <el-input v-model="categoryKeyword" :placeholder="t('training.searchCategories')"
            class="easy-input" style="width: 260px;" clearable @clear="loadCategoryTable"
            @keyup.enter="loadCategoryTable" />
          <el-button class="easy-button" @click="loadCategoryTable">{{ t('common.search') }}</el-button>
        </div>
        <div class="toolbar-right">
          <el-button link size="default" @click="openCreateCategory">
            <el-icon :size="16"><Plus /></el-icon>
            {{ t('training.addCategory') }}
          </el-button>
        </div>
      </div>

      <el-table :data="categoryTableData" class="easy-table" size="default">
        <el-table-column prop="id" :label="t('common.id')" width="80" />
        <el-table-column prop="name" :label="t('training.categoryName')" min-width="200" />
        <el-table-column prop="description" :label="t('training.description')" min-width="200">
          <template #default="scope">
            <el-tooltip placement="bottom" :content="scope.row.description" :show-after="200">
              <span class="text-ellipsis">{{ scope.row.description }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="sampleCount" :label="t('training.sampleCount')" width="120" align="center" />
        <el-table-column :label="t('common.created')" width="220">
          <template #default="scope">{{ formatDateTime(scope.row.createdAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="160">
          <template #default="scope">
            <el-button type="primary" link size="small" @click="openEditCategory(scope.row)">
              <el-icon :size="16"><Edit /></el-icon>
            </el-button>
            <el-button type="danger" link size="small" @click="handleDeleteCategory(scope.row)">
              <el-icon :size="16"><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-container">
        <el-pagination v-model:current-page="categoryPage" v-model:page-size="categoryPageSize"
          :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next"
          :total="categoryTotal" @size-change="handleCategorySizeChange"
          @current-change="handleCategoryPageChange" class="easy-pagination" />
      </div>

      <template #footer>
        <el-button class="easy-button" @click="showCategoryDialog = false">{{ t('common.close') }}</el-button>
      </template>
    </el-dialog>

    <!-- Category Form Dialog (Create/Edit) -->
    <el-dialog v-model="showCategoryForm" :title="categoryForm.id ? t('training.editCategory') : t('training.addCategory')"
      width="520px" class="easy-dialog" align-center :close-on-click-modal="false">
      <el-form :model="categoryForm" label-width="150px" class="easy-form">
        <el-form-item :label="t('training.categoryName')" required>
          <el-input v-model="categoryForm.name" :placeholder="t('training.categoryNamePlaceholder')"
            class="easy-input" />
        </el-form-item>
        <el-form-item :label="t('training.description')">
          <el-input v-model="categoryForm.description" type="textarea" :rows="3"
            :placeholder="t('training.descriptionPlaceholder')" class="easy-input" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button class="easy-button" @click="showCategoryForm = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" class="easy-button" @click="handleSaveCategory">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Edit, Delete } from '@element-plus/icons-vue'
import { formatDateTime } from '@/utils/times'
import {
  trainingApi,
  type SampleCategory
} from '../../api/training'

interface Props {
  categoryList: SampleCategory[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'category-updated': []
}>()

const { t } = useI18n()

const showCategoryDialog = ref(false)
const categoryTableData = ref<SampleCategory[]>([])
const categoryPage = ref(1)
const categoryPageSize = ref(20)
const categoryTotal = ref(0)
const categoryKeyword = ref('')

watch(showCategoryDialog, async (val) => {
  if (val) {
    categoryPage.value = 1
    categoryKeyword.value = ''
    await loadCategoryTable()
  }
})

const showCategoryForm = ref(false)
const categoryForm = reactive({
  id: 0,
  name: '',
  description: ''
})

async function loadCategories() {
  try {
    const res = await trainingApi.listSampleCategories({ page: 1, pageSize: 1000 })
    if (res.code === 0 && res.data) {
      if (!props.categoryList.length && res.data.length) {
        emit('category-updated')
      }
    }
  } catch {
    // ignore
  }
}

async function loadCategoryTable() {
  try {
    const res = await trainingApi.listSampleCategories({
      keyword: categoryKeyword.value || undefined,
      page: categoryPage.value,
      pageSize: categoryPageSize.value
    })
    if (res.code === 0) {
      categoryTableData.value = res.data || []
      categoryTotal.value = (res.meta as any)?.total ?? 0
    }
  } catch {
    ElMessage.error(t('common.loadFailed'))
  }
}

function openCreateCategory() {
  categoryForm.id = 0
  categoryForm.name = ''
  categoryForm.description = ''
  showCategoryForm.value = true
}

function openEditCategory(row: SampleCategory) {
  categoryForm.id = row.id
  categoryForm.name = row.name
  categoryForm.description = row.description
  showCategoryForm.value = true
}

async function handleSaveCategory() {
  const name = categoryForm.name.trim()
  if (!name) {
    ElMessage.warning(t('training.categoryNameRequired'))
    return
  }
  try {
    if (categoryForm.id > 0) {
      const res = await trainingApi.updateSampleCategory(categoryForm.id, {
        name,
        description: categoryForm.description
      })
      if (res.code !== 0) {
        ElMessage.error(res.message || t('common.operationFailed'))
        return
      }
      ElMessage.success(t('training.categoryUpdated'))
    } else {
      const res = await trainingApi.createSampleCategory({
        name,
        description: categoryForm.description
      })
      if (res.code !== 0) {
        ElMessage.error(res.message || t('common.operationFailed'))
        return
      }
      ElMessage.success(t('training.categoryCreated'))
    }
    showCategoryForm.value = false
    await loadCategories()
    await loadCategoryTable()
    emit('category-updated')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
  }
}

async function handleDeleteCategory(row: SampleCategory) {
  const msg = row.sampleCount > 0
    ? t('training.deleteCategoryWithSamplesConfirm')
    : t('training.deleteCategoryConfirm')
  try {
    await ElMessageBox.confirm(msg, t('common.confirm'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    const res = await trainingApi.deleteSampleCategory(row.id)
    if (res.code !== 0) {
      ElMessage.error(res.message || t('common.operationFailed'))
      return
    }
    ElMessage.success(t('training.categoryDeleted'))
    await loadCategories()
    await loadCategoryTable()
    emit('category-updated')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
    }
  }
}

const handleCategorySizeChange = (size: number) => {
  categoryPageSize.value = size
  categoryPage.value = 1
  loadCategoryTable()
}

const handleCategoryPageChange = (p: number) => {
  categoryPage.value = p
  loadCategoryTable()
}

onMounted(() => {
  loadCategories()
})
</script>

<style scoped>
.category-dialog {
  display: inline-block;
}
</style>

<style>
.category-management-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 15px;
  margin-bottom: 20px;
  background: var(--surface-light);
  border-radius: 8px;
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

.text-ellipsis {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}
</style>