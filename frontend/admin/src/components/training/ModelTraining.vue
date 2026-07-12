<template>
  <div class="model-training">
    <!-- Model name -->
    <div class="mt-field">
      <label class="mt-label">{{ t('training.modelName') }}</label>
      <div class="mt-model-wrap">
        <el-input
          v-model="trainingForm.modelName"
          :placeholder="t('training.modelNamePlaceholder')"
          class="mt-model-name easy-input"
          maxlength="64"
        />
        <div v-if="modelNameError" class="mt-error">{{ modelNameError }}</div>
      </div>
    </div>

    <!-- Parameters -->
    <section class="mt-section">
      <header class="mt-section__head">
        <span class="mt-section__title">{{ t('training.modelParams') }}</span>
        <div class="mt-presets">
          <el-button size="small" @click="applyPreset('quick')">{{ t('training.presetQuick') }}</el-button>
          <el-button size="small" @click="applyPreset('balanced')">{{ t('training.presetBalanced') }}</el-button>
          <el-button size="small" @click="applyPreset('accurate')">{{ t('training.presetAccurate') }}</el-button>
        </div>
      </header>
      <div class="mt-params">
        <div class="mt-param">
          <label>{{ t('training.learningRate') }}</label>
          <el-input-number
            v-model="trainingForm.params.learningRate"
            class="easy-input"
            :min="0.01"
            :max="1"
            :step="0.01"
            :precision="2"
            controls-position="right"
          />
        </div>
        <div class="mt-param">
          <label>{{ t('training.epoch') }}</label>
          <el-input-number class="easy-input" v-model="trainingForm.params.epoch" :min="1" :max="100" :step="1" controls-position="right" />
        </div>
        <div class="mt-param">
          <label>{{ t('training.wordNgrams') }}</label>
          <el-input-number class="easy-input" v-model="trainingForm.params.wordNgrams" :min="1" :max="5" :step="1" controls-position="right" />
        </div>
        <div class="mt-param">
          <label>{{ t('training.dim') }}</label>
          <el-input-number class="easy-input" v-model="trainingForm.params.dim" :min="10" :max="300" :step="10" controls-position="right" />
        </div>
        <div class="mt-param">
          <label>{{ t('training.loss') }}</label>
          <el-select v-model="trainingForm.params.loss" class="mt-loss">
            <el-option label="softmax" value="softmax" />
            <el-option label="ns" value="ns" />
            <el-option label="hs" value="hs" />
          </el-select>
        </div>
      </div>
    </section>

    <!-- Target classes -->
    <section class="mt-section">
      <header class="mt-section__head">
        <div>
          <span class="mt-section__title">{{ t('training.targetClasses') }}</span>
          <span class="mt-section__meta">{{ classes.length }}</span>
        </div>
        <el-button class="easy-button" text size="small"  @click="addClass">
          <el-icon><Plus /></el-icon>{{ t('training.addClass') }}
        </el-button>
      </header>
      <p class="mt-section__hint">{{ t('training.trainingGuide') }}</p>

      <div v-if="classError" class="mt-error mt-error--block">{{ classError }}</div>

      <div class="mt-class-list">
        <div
          v-for="(cls, idx) in classes"
          :key="idx"
          class="mt-class-row"
          :class="{ 'mt-class-row--invalid': classErrors[idx] }"
        >
          <div class="mt-class-row__top">
            <span class="mt-class-row__idx">{{ idx + 1 }}</span>
            <el-button
              class="mt-class-row__remove"
              text
              type="danger"
              :disabled="classes.length <= 1"
              @click="removeClass(idx)"
            >
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <div class="mt-class-row__name">
            <el-input
              v-model="cls.name"
              size="default"
              :placeholder="t('training.classNamePlaceholder')"
              class="mt-class-input easy-input"
            />
            <div v-if="classErrors[idx]" class="mt-error">{{ classErrors[idx] }}</div>
          </div>

          <div class="mt-class-row__sources">
            <div class="mt-sources__label">{{ t('training.sourceSamples') }}</div>
            <div
              v-for="(grp, gidx) in cls.sources"
              :key="gidx"
              class="mt-source-group"
            >
              <el-select
                v-model="grp.category"
                :placeholder="t('training.sourceCategory')"
                class="mt-src-cat"
                @change="onCategoryChange(cls, gidx)"
              >
                <el-option
                  v-for="g in sourceGroups"
                  :key="g.category"
                  :label="g.category"
                  :value="g.category"
                />
              </el-select>
              <el-select
                v-model="grp.tags"
                multiple
                filterable
                clearable
                :placeholder="t('training.sourceTagSelectPlaceholder')"
                class="mt-src-tags"
              >
                <el-option
                  v-for="tg in tagsForCategory(grp.category)"
                  :key="tg.tag"
                  :label="tg.tag"
                  :value="tg.tag"
                />
              </el-select>
              <el-select
                v-model="grp.limitType"
                :placeholder="t('training.sourceLimit')"
                class="mt-src-limit"
              >
                <el-option :label="t('training.limitUnlimited')" value="unlimited" />
                <el-option :label="t('training.limitRandom')" value="random" />
                <el-option :label="t('training.limitFirst')" value="first" />
                <el-option :label="t('training.limitLast')" value="last" />
                <el-option :label="t('training.limitMiddle')" value="middle" />
              </el-select>
              <el-input-number
                v-if="grp.limitType !== 'unlimited'"
                v-model="grp.limitN"
                :min="1"
                :max="100000"
                :step="1"
                controls-position="right"
                class="mt-src-n"
              />
              <el-button
                class="mt-src-remove"
                text
                type="danger"
                :disabled="cls.sources.length <= 1"
                @click="removeClassSource(cls, gidx)"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <el-button text size="small" class="mt-add-group" @click="addClassSource(cls)">
              <el-icon><Plus /></el-icon>{{ t('training.addSourceGroup') }}
            </el-button>
          </div>
        </div>
      </div>

      <div v-if="classes.length === 0" class="mt-empty">{{ t('training.noClasses') }}</div>
    </section>


    <!-- Actions -->
    <div class="mt-actions">
      <el-button
        type="success"
        :loading="trainingRunning"
        :disabled="trainingRunning"
        @click="handleStartTraining"
      >
      {{ t('training.startTraining') }}
      </el-button>
    </div>

    <!-- Status panel -->
    <section v-if="currentTrainingTask" class="mt-status">
      <div class="mt-status__head">
        <div class="mt-status__left">
          <span class="mt-status__label">{{ t('training.trainStatusLabel') }}:</span>
          <el-tag v-if="currentTrainingTask.status === 'running'" type="warning" size="small">{{ t('training.trainStatusRunning') }}</el-tag>
          <el-tag v-else-if="currentTrainingTask.status === 'completed'" type="success" size="small">{{ t('training.trainStatusCompleted') }}</el-tag>
          <el-tag v-else-if="currentTrainingTask.status === 'failed'" type="danger" size="small">{{ t('training.trainStatusFailed') }}</el-tag>
          <el-tag v-else type="info" size="small">{{ t('training.trainStatusPending') }}</el-tag>
          <span v-if="currentTrainingTask.modelId" class="mt-status__version">
            {{ t('training.version') }}: v{{ currentTrainingTask.modelId }}
          </span>
        </div>
        <el-button text size="small" :loading="trainLogRefreshing" @click="handleRefreshTrainLog">
          <el-icon><Refresh /></el-icon>{{ t('common.refresh') }}
        </el-button>
      </div>
      <div class="mt-status__log-label">{{ t('training.trainLogLabel') }}</div>
      <el-input
        type="textarea"
        :rows="10"
        readonly
        class="mt-log"
        :model-value="currentTrainingTask.trainResult"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Refresh } from '@element-plus/icons-vue'
import { trainingApi, type TrainingTask, type CategoryMapping, type SampleCategory } from '@/api/training'

const { t } = useI18n()

// ========== Form state ==========
const trainingForm = reactive({
  modelName: '',
  params: {
    learningRate: 0.1,
    epoch: 5,
    wordNgrams: 2,
    dim: 100,
    loss: 'softmax'
  }
})

// Target classes: each maps a name to one or more source sample groups.
interface SourceGroupForm {
  category: string
  tags: string[]
  limitType: 'unlimited' | 'random' | 'first' | 'last' | 'middle'
  limitN: number
}
const classes = ref<{ name: string; sources: SourceGroupForm[] }[]>([])

function makeSourceGroup(): SourceGroupForm {
  return { category: '', tags: [], limitType: 'unlimited', limitN: 100 }
}

// Tag pool grouped by source category. Categories come from the authoritative
// category list (so they show even when no samples exist yet), and tags are
// loaded per category.
const sourceGroups = ref<{ category: string; categoryId: number; tags: { tag: string; count: number }[] }[]>([])

// API / polling state
const currentTrainingTask = ref<TrainingTask | null>(null)
const trainingRunning = ref(false)
const trainLogRefreshing = ref(false)
let trainingPollTimer: ReturnType<typeof setInterval> | null = null

// Presets
const presetConfigs: Record<string, { learningRate: number; epoch: number; wordNgrams: number; dim: number }> = {
  quick: { learningRate: 0.1, epoch: 5, wordNgrams: 1, dim: 50 },
  accurate: { learningRate: 0.01, epoch: 20, wordNgrams: 3, dim: 150 },
  balanced: { learningRate: 0.05, epoch: 10, wordNgrams: 2, dim: 100 }
}

// ========== Validation (runs on submit, then shown inline) ==========
const validated = ref(false)

const classErrors = computed<Record<number, string>>(() => {
  const errs: Record<number, string> = {}
  if (!validated.value) return errs
  const seen = new Set<string>()
  classes.value.forEach((cls, idx) => {
    const name = cls.name.trim()
    if (!name) {
      errs[idx] = t('training.classNameEmpty')
    } else if (!/^[a-z]{1,10}$/.test(name)) {
      errs[idx] = t('training.classNameFormat')
    } else if (seen.has(name)) {
      errs[idx] = t('training.classNameDuplicate')
    } else {
      seen.add(name)
    }
  })
  return errs
})

// Model name error is shown inline right under the input, not in the class block.
const modelNameError = computed<string>(() => {
  if (!validated.value) return ''
  if (!trainingForm.modelName.trim()) return t('training.modelNameRequiredHint')
  return ''
})

const classError = computed<string>(() => {
  if (!validated.value) return ''
  if (classes.value.length < 2) return t('training.minTwoClasses')
  if (classes.value.some((c) => !c.sources.some((g) => g.tags.length > 0))) return t('training.noTagsHint')
  if (Object.keys(classErrors.value).length > 0) return t('training.fixClassName')
  return ''
})

// Combined error used for the submit-time blocking warning.
const formError = computed<string>(() => modelNameError.value || classError.value)

// ========== Actions ==========
function addClass() {
  classes.value.push({ name: '', sources: [makeSourceGroup()] })
}

function removeClass(idx: number) {
  if (classes.value.length <= 1) return
  classes.value.splice(idx, 1)
}

function addClassSource(cls: { name: string; sources: SourceGroupForm[] }) {
  cls.sources.push(makeSourceGroup())
}

function removeClassSource(cls: { name: string; sources: SourceGroupForm[] }, gidx: number) {
  if (cls.sources.length <= 1) return
  cls.sources.splice(gidx, 1)
}

// Tags available under a chosen category.
function tagsForCategory(category: string) {
  return sourceGroups.value.find((g) => g.category === category)?.tags || []
}

// Switching category invalidates the previously selected tags.
function onCategoryChange(cls: { name: string; sources: SourceGroupForm[] }, gidx: number) {
  cls.sources[gidx].tags = []
}


function applyPreset(name: string) {
  const cfg = presetConfigs[name]
  if (!cfg) return
  trainingForm.params.learningRate = cfg.learningRate
  trainingForm.params.epoch = cfg.epoch
  trainingForm.params.wordNgrams = cfg.wordNgrams
  trainingForm.params.dim = cfg.dim
}

// ========== Tag pool ==========
async function loadTagPool() {
  try {
    const catRes = await trainingApi.listSampleCategories({ pageSize: 1000 })
    const cats: SampleCategory[] = catRes.code === 0 && Array.isArray(catRes.data) ? catRes.data : []
    const groups = await Promise.all(
      cats.map(async (c) => {
        let tags: { tag: string; count: number }[] = []
        try {
          const tagRes = await trainingApi.listTags(c.id)
          if (tagRes.code === 0 && Array.isArray(tagRes.data)) {
            tags = tagRes.data.map((t: string) => ({ tag: t, count: 0 }))
          }
        } catch {
          // ignore; tags are optional for a category
        }
        return { category: c.name, categoryId: c.id, tags }
      })
    )
    sourceGroups.value = groups
  } catch {
    // ignore; tag pool is optional for starting training
  }
}

// ========== Training ==========
async function handleStartTraining() {
  if (trainingRunning.value) return
  validated.value = true
  if (formError.value) {
    ElMessage.warning(formError.value)
    return
  }
  const confirmMsg = currentTrainingTask.value ? t('training.retrainConfirm') : t('training.trainConfirm')
  try {
    await ElMessageBox.confirm(confirmMsg, t('common.confirm'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
  } catch {
    return
  }

  const sampleMappings: CategoryMapping[] = classes.value.map((c) => ({
    targetClass: c.name.trim(),
    sources: c.sources.map((g) => ({
      category: g.category,
      tags: [...g.tags],
      limitType: g.limitType,
      limitN: g.limitN
    }))
  }))

  trainingRunning.value = true
  try {
    const res = await trainingApi.startTraining({
      modelName: trainingForm.modelName.trim(),
      algorithm: 'FastText',
      params: { ...trainingForm.params },
      sampleMappings
    })
    if (res.code !== 0) {
      ElMessage.error(res.message || t('common.operationFailed'))
      return
    }
    ElMessage.success(t('training.trainStarted'))
    currentTrainingTask.value = res.data
    startTrainingPoll()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.message || e.message || t('common.operationFailed'))
  } finally {
    trainingRunning.value = false
  }
}

function stopTrainingPoll() {
  if (trainingPollTimer) {
    clearInterval(trainingPollTimer)
    trainingPollTimer = null
  }
}

async function refreshTrainingTask() {
  if (!currentTrainingTask.value) return
  try {
    const res = await trainingApi.getTraining(currentTrainingTask.value.id)
    if (res.code === 0 && res.data) {
      currentTrainingTask.value = res.data
      if (res.data.status === 'running') {
        startTrainingPoll()
      } else {
        stopTrainingPoll()
        if (res.data.modelId) ElMessage.success(t('training.versionCreated'))
        if (res.data.status === 'completed') ElMessage.success(t('training.trainCompleted'))
        else if (res.data.status === 'failed') ElMessage.error(t('training.trainFailed'))
      }
    }
  } catch {
    // ignore
  }
}

function startTrainingPoll() {
  if (trainingPollTimer) return
  trainingPollTimer = setInterval(async () => {
    await refreshTrainingTask()
  }, 2000)
}

async function handleRefreshTrainLog() {
  if (!currentTrainingTask.value) return
  trainLogRefreshing.value = true
  try {
    await refreshTrainingTask()
  } finally {
    trainLogRefreshing.value = false
  }
}

// ========== Lifecycle ==========
onMounted(async () => {
  await loadTagPool()
})

onBeforeUnmount(() => {
  stopTrainingPoll()
})

// Seed one starter class row for convenience.
watch(
  () => classes.value.length,
  (len) => {
    if (len === 0) addClass()
  },
  { immediate: true }
)
</script>

<style scoped lang="scss">
.model-training {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.mt-field {
  display: flex;
  align-items: flex-start;
  gap: 12px;

  .mt-label {
    width: 96px;
    flex-shrink: 0;
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
    line-height: 32px;
  }

  .mt-model-wrap {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .mt-model-name {
    max-width: 360px;
  }
}

.mt-section {
  background: var(--surface-light);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  padding: 16px 18px;

  &__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 6px;
  }

  &__title {
    font-size: 14px;
    font-weight: 600;
    color: var(--foreground);
  }

  &__meta {
    margin-left: 8px;
    font-size: 12px;
    color: var(--foreground-muted);
    background: var(--surface);
    border: 1px solid var(--border-default);
    border-radius: 10px;
    padding: 1px 8px;
  }

  &__hint {
    margin: 0 0 12px;
    font-size: 12px;
    color: var(--foreground-muted);
  }
}

.mt-presets {
  display: flex;
  gap: 8px;
}

.mt-class-list {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 12px;

  @media (min-width: 640px) {
    grid-template-columns: repeat(1, minmax(0, 1fr));
  }

  @media (min-width: 960px) {
    grid-template-columns: repeat(1, minmax(0, 1fr));
  }

  @media (min-width: 1280px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.mt-class-row {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  padding: 10px 12px;
  background: var(--surface);
  border: 1px solid var(--border-default);
  border-radius: 8px;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5, #a0cfff);
  }

  &--invalid {
    border-color: var(--el-color-danger, #f56c6c);
  }

  &__top {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__idx {
    font-size: 13px;
    font-weight: 600;
    color: var(--foreground-muted);
  }


  &__name {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  &__remove {
    margin-top: 0;
  }

  &__sources {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding-top: 6px;
    margin-top: 2px;
    border-top: 1px dashed var(--border-default);
  }
}

.mt-sources__label {
  font-size: 12px;
  font-weight: 500;
  color: var(--foreground-muted);
}

.mt-source-group {
  display: grid;
  grid-template-columns: 130px 1fr 110px 92px 30px;
  gap: 6px;
  align-items: center;

  @media (max-width: 560px) {
    grid-template-columns: 1fr 1fr;
  }
}

.mt-src-cat,
.mt-src-tags,
.mt-src-limit,
.mt-src-n {
  width: 100%;
}

.mt-add-group {
  align-self: flex-start;
}

.mt-class-input {
  :deep(.el-input__wrapper) {
    border-radius: 6px;
  }
}

.mt-params {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 12px 18px;
}

.mt-param {
  display: flex;
  flex-direction: column;
  gap: 6px;

  label {
    font-size: 12px;
    color: var(--foreground-muted);
  }

  :deep(.el-input-number),
  :deep(.el-select) {
    width: 100%;
  }
}

.mt-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;

  &__hint {
    font-size: 12px;
    color: var(--el-color-warning, #e6a23c);
  }
}

.mt-error {
  font-size: 12px;
  color: var(--el-color-danger, #f56c6c);

  &--block {
    margin-bottom: 10px;
  }
}

.mt-empty {
  font-size: 13px;
  color: var(--foreground-muted);
  padding: 8px 0;
}

.mt-status {
  background: var(--surface-light);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  padding: 16px 18px;

  &__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  &__left {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  &__label {
    font-size: 13px;
    color: var(--foreground-muted);
  }

  &__version {
    font-size: 13px;
    color: var(--foreground-muted);
  }

  &__log-label {
    margin: 0 0 8px;
    font-size: 13px;
    color: var(--foreground-muted);
  }
}

.mt-log {
  background: var(--surface);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  resize: none;
}
</style>
