<template>
  <div class="training-page">
    <el-card class="glass-card main-card">
      <el-tabs v-model="activeTab" class="training-tabs" @tab-change="handleTabChange">
        <!-- Tab 1: Sample Management -->
        <el-tab-pane :label="t('training.sampleManagement')" name="samples">
          <div class="tab-content">
            <SampleManagement
              v-if="activeTab === 'samples'"
              :category-list="categoryList"
              :tags-map="tagCache"
              @sample-updated="loadSamples"
              @category-updated="loadCategories"
              @tags-loaded="onTagsLoaded"
            />
          </div>
        </el-tab-pane>

        <!-- Tab 2: Model Training -->
        <el-tab-pane :label="t('training.modelTraining')" name="training">
          <div class="tab-content">
            <ModelTraining
              v-if="activeTab === 'training'"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
</div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { trainingApi } from '@/api/training'
import type { SampleCategory } from '@/api/training'
import SampleManagement from '../components/training/SampleManagement.vue'
import ModelTraining from '../components/training/ModelTraining.vue'

const { t } = useI18n()

// ========== Active Tab ==========
const activeTab = ref('samples')

// ========== Tab 1: Sample Management ==========

// Categories needed for SampleManagement and ModelTraining
const categoryList = ref<SampleCategory[]>([])
const tagCache = ref<Record<number, string[]>>({})

// Sample filter state
const sampleFilter = reactive({
  categoryId: undefined as number | undefined,
  tag: '',
  keyword: ''
})

// No longer maintaining separate deploy state — removed.

// Load categories for both tabs
async function loadCategories() {
  try {
    const res = await trainingApi.listSampleCategories({ page: 1, pageSize: 1000 })
    if (res.code === 0 && res.data) {
      categoryList.value = res.data
      if (!sampleFilter.categoryId && categoryList.value.length > 0) {
        sampleFilter.categoryId = categoryList.value[0].id
      }
    } else {
      sampleFilter.categoryId = undefined
    }
  } catch {}
}

function onTagsLoaded(categoryId: number, tags: string[]) {
  tagCache.value = { ...tagCache.value, [categoryId]: tags }
}

function handleTabChange() {
  // Each tab component handles its own data loading
}

watch(activeTab, () => {
  if (activeTab.value !== 'training') {
    console.log('Training tab deactivated')
  }
})

async function loadSamples() {
  try {
    const res = await trainingApi.listSamples({
      categoryId: sampleFilter.categoryId || undefined,
      tag: sampleFilter.tag || undefined,
      keyword: sampleFilter.keyword || undefined,
      page: 1,
      pageSize: 1000
    })
    if (res.code === 0) {
      // Samples are loaded in SampleManagement component
    }
  } catch {}
}

onMounted(async () => {
  await loadCategories()
  await loadSamples()
})
</script>

<script lang="ts">
export default {
  name: 'Training'
}
</script>

<style scoped lang="scss">
.training-page {
  height: 100%;
  margin: 0 auto;
}

.main-card {
  min-height: calc(100vh - 140px);
}

.main-card :deep(.el-card__body) {
  padding: 20px 25px 28px;
}

.training-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.training-tabs :deep(.el-tabs__header) {
  margin: 0 0 16px;
  padding: 0;
  border-bottom: 1px solid var(--border-default);
}

.training-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
}

.training-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow-y: auto;
}

.tab-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding-top: 20px;
}

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