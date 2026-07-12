<template>
  <div class="cond-node" :class="{ 'cond-node--nested': depth > 0 }">
    <!-- 逻辑组：and / or / not -->
    <template v-if="isGroup">
      <div class="cond-row cond-row--group">
        <el-select v-model="groupOp" class="easy-select cond-op-select">
          <el-option :label="t('FilterRules.conditionAnd')" value="and" />
          <el-option :label="t('FilterRules.conditionOr')" value="or" />
          <el-option :label="t('FilterRules.conditionNot')" value="not" />
        </el-select>
      </div>

      <div v-if="groupOp === 'not'" class="cond-not-box">
        <div class="cond-subtitle">{{ t('FilterRules.conditionNotHint') }}</div>
        <ConditionNodeEditor
          v-if="notChild"
          :model-value="notChild"
          :features="features"
          :depth="depth + 1"
          @update:model-value="updateNotChild"
        />
        <el-button v-else type="primary" link class="easy-button ghost small-button" @click="ensureNotChild">{{ t('FilterRules.addInnerCondition') }}</el-button>
      </div>

      <div v-else class="cond-children">
        <div v-for="(ch, i) in groupChildren" :key="i" class="cond-child-card">
          <div class="cond-child-head">
            <span class="cond-child-idx">{{ t('FilterRules.condition') }} {{ i + 1 }}</span>
            <el-button type="danger" link size="small" class="easy-button ghost small-button" @click="removeChild(i)">{{ t('common.delete') }}</el-button>
          </div>
          <ConditionNodeEditor
            :model-value="ch"
            :features="features"
            :depth="depth + 1"
            @update:model-value="(v) => updateChild(i, v)"
          />
        </div>
        <el-dropdown trigger="click" @command="onAddChild">
          <el-button type="primary" plain size="small" class="easy-button ghost small-button cond-add-btn">
            <el-icon><Plus /></el-icon>
            {{ t('FilterRules.addCondition') }}
          </el-button>
          <template #dropdown>
            <el-dropdown-menu class="easy-dropdown-menu">
              <el-dropdown-item command="cmp">{{ t('FilterRules.conditionCmp') }}</el-dropdown-item>
              <el-dropdown-item command="feat">{{ t('FilterRules.conditionFeat') }}</el-dropdown-item>
              <el-dropdown-item command="group" :disabled="depth >= maxDepth">
                {{ t('FilterRules.conditionGroup') }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </template>

    <!-- 叶子：feat -->
    <template v-else-if="leafOp === 'feat'">
      <div class="cond-row cond-row--wrap">
        <span class="cond-label">{{ t('FilterRules.feature') }}</span>
        <el-select v-model="featKey" :placeholder="t('FilterRules.selectFeature')" class="easy-select cond-field-mid" filterable>
          <el-option
            v-for="f in features"
            :key="f.featureKey"
            :label="`${f.label} (${f.featureKey})`"
            :value="f.featureKey"
          />
        </el-select>
        <el-select v-model="featKind" class="easy-select cond-field-sm">
          <el-option :label="t('FilterRules.conditionTrue')" value="true" />
          <el-option :label="t('FilterRules.conditionFalse')" value="false" />
        </el-select>
      </div>
      <p class="cond-hint">{{ t('FilterRules.conditionFeatHint') }}</p>
    </template>

    <!-- 叶子：cmp -->
    <template v-else-if="leafOp === 'cmp'">
      <div class="cond-row cond-row--wrap">
        <span class="cond-label">{{ t('FilterRules.feature') }}</span>
        <el-select v-model="cmpFeature" :placeholder="t('FilterRules.feature')" class="easy-select cond-field-mid" filterable>
          <el-option
            v-for="f in features"
            :key="f.featureKey"
            :label="`${f.label} (${f.featureKey})`"
            :value="f.featureKey"
          />
        </el-select>
        <el-select v-model="cmpKind" class="easy-select cond-field-sm">
          <el-option v-for="k in cmpKinds" :key="k.value" :label="k.label" :value="k.value" />
        </el-select>
        <el-input-number v-model="cmpValue" :controls="true" class="easy-input-number cond-field-num" />
      </div>
    </template>

    <template v-else>
      <el-alert type="warning" :closable="false" show-icon :title="t('FilterRules.unknownConditionNode')">
        <template #default>
          <span>{{ t('FilterRules.unknownConditionNodeHint') }}</span>
        </template>
      </el-alert>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FilterFeature } from '../../api/filter.ts'
import type { CondNode } from './condTypes.ts'
import { CMP_KINDS } from './condTypes.ts'
import ConditionNodeEditor from './ConditionNodeEditor.vue'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue: CondNode
    features: FilterFeature[]
    depth?: number
  }>(),
  { depth: 0 }
)

const emit = defineEmits<{
  'update:modelValue': [value: CondNode]
}>()

const cmpKinds = CMP_KINDS
const maxDepth = 12

const isGroup = computed(() => {
  const o = (props.modelValue?.op || '').toLowerCase()
  return o === 'and' || o === 'or' || o === 'not'
})

const leafOp = computed(() => (props.modelValue?.op || '').toLowerCase())

const groupOp = computed({
  get: () => (props.modelValue?.op || 'and').toLowerCase() as 'and' | 'or' | 'not',
  set: (v) => {
    patchGroupOp(v)
  }
})

const groupChildren = computed(() => {
  if (!isGroup.value) return []
  const o = groupOp.value
  if (o === 'not') return []
  return props.modelValue.children || []
})

const notChild = computed(() => {
  if (groupOp.value !== 'not') return null
  const ch = props.modelValue.children || []
  return ch[0] || null
})

function firstFeatureKey() {
  return props.features[0]?.featureKey || ''
}

function emitNode(n: CondNode) {
  emit('update:modelValue', n)
}

function patchGroupOp(v: 'and' | 'or' | 'not') {
  const cur = props.modelValue
  const curOp = (cur.op || '').toLowerCase()

  if (v === 'not') {
    let inner: CondNode
    if (curOp === 'not') {
      inner = cur.children?.[0] || makeCmpDefault()
    } else if (curOp === 'and' || curOp === 'or') {
      const ch = cur.children || []
      if (ch.length === 0) inner = makeCmpDefault()
      else if (ch.length === 1) inner = ch[0]!
      else inner = { op: curOp, children: [...ch] }
    } else {
      inner = { ...cur, children: undefined }
    }
    emitNode({ op: 'not', children: [inner] })
    return
  }

  let prevChildren: CondNode[] = []
  if (curOp === 'not') {
    const c = cur.children?.[0]
    if (c) prevChildren = [c]
  } else {
    prevChildren = [...(cur.children || [])]
  }
  emitNode({ op: v, children: prevChildren })
}

function updateChild(i: number, n: CondNode) {
  const op = groupOp.value
  const ch = [...(props.modelValue.children || [])]
  ch[i] = n
  emitNode({ op, children: ch })
}

function removeChild(i: number) {
  const op = groupOp.value
  const ch = [...(props.modelValue.children || [])]
  ch.splice(i, 1)
  emitNode({ op, children: ch })
}

function updateNotChild(n: CondNode) {
  emitNode({ op: 'not', children: [n] })
}

function ensureNotChild() {
  emitNode({ op: 'not', children: [makeCmpDefault()] })
}

function onAddChild(cmd: string) {
  if (cmd === 'group' && props.depth >= maxDepth) return
  const op = groupOp.value
  const ch = [...(props.modelValue.children || [])]
  if (cmd === 'cmp') ch.push(makeCmpDefault())
  else if (cmd === 'feat') ch.push(makeFeatDefault())
  else if (cmd === 'group') ch.push(makeGroupDefault())
  emitNode({ op, children: ch })
}

function makeCmpDefault(): CondNode {
  return {
    op: 'cmp',
    feature: firstFeatureKey(),
    kind: 'gt',
    value: 0
  }
}

function makeFeatDefault(): CondNode {
  return { op: 'feat', feature: firstFeatureKey(), kind: 'true' }
}

function makeGroupDefault(): CondNode {
  return { op: 'and', children: [] }
}

const featKey = computed({
  get: () => props.modelValue.feature || firstFeatureKey(),
  set: (v) => emitNode({ op: 'feat', feature: v, kind: featKind.value })
})

const featKind = computed({
  get: () => props.modelValue.kind || 'true',
  set: (v) => emitNode({ op: 'feat', feature: featKey.value, kind: v })
})

const cmpFeature = computed({
  get: () => props.modelValue.feature || firstFeatureKey(),
  set: (v) => {
    emitNode({
      op: 'cmp',
      feature: v,
      kind: cmpKind.value,
      value: cmpValue.value
    })
  }
})

const cmpKind = computed({
  get: () => (props.modelValue.kind || 'gt').toLowerCase(),
  set: (v) => {
    emitNode({
      op: 'cmp',
      feature: cmpFeature.value,
      kind: v,
      value: cmpValue.value
    })
  }
})

const cmpValue = computed({
  get: () => (typeof props.modelValue.value === 'number' ? props.modelValue.value : Number(props.modelValue.value) || 0),
  set: (v) => {
    const n = typeof v === 'number' ? v : 0
    emitNode({
      op: 'cmp',
      feature: cmpFeature.value,
      kind: cmpKind.value,
      value: n
    })
  }
})

</script>

<style scoped>
.cond-node {
  border-radius: 8px;
}
.cond-node--nested {
  padding: 10px 12px;
  border: 1px solid var(--border-default);
  background: var(--surface-light);
}
.cond-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}
.cond-row--wrap {
  flex-wrap: wrap;
}
.cond-row--group {
  margin-bottom: 12px;
}
.cond-label {
  flex-shrink: 0;
  width: 72px;
  font-size: 13px;
  color: var(--foreground-muted);
}
.cond-op-select {
  width: 220px;
}
.cond-field-grow {
  flex: 1;
  min-width: 200px;
}
.cond-field-mid {
  flex: 1;
  min-width: 180px;
}
.cond-field-sm {
  width: 140px;
}
.cond-field-num {
  width: 160px;
}
.cond-hint {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--foreground-muted);
  line-height: 1.45;
}
.cond-not-box {
  margin-top: 8px;
}
.cond-subtitle {
  font-size: 12px;
  color: var(--foreground-muted);
  margin-bottom: 8px;
}
.cond-children {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
}
.cond-child-card {
  padding: 8px 0;
  border-top: 1px dashed var(--border-default);
}
.cond-child-card:first-of-type {
  border-top: none;
  padding-top: 0;
}
.cond-child-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.cond-child-idx {
  font-size: 12px;
  font-weight: 600;
  color: var(--foreground);
}
.cond-add-btn {
  align-self: flex-start;
  margin-top: 4px;
}
</style>
