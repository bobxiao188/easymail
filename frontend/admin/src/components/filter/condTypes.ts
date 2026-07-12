/** 与后端 scanner 条件 AST（CondNode）结构一致 */
export interface CondNode {
  op: string
  feature?: string
  kind?: string
  value?: number
  children?: CondNode[]
}

export const CMP_KINDS = [
  { value: 'eq', label: '等于 (=)' },
  { value: 'ne', label: '不等于 (≠)' },
  { value: 'gt', label: '大于 (>)' },
  { value: 'ge', label: '大于等于 (≥)' },
  { value: 'lt', label: '小于 (<)' },
  { value: 'le', label: '小于等于 (≤)' }
] as const

export function defaultRoot(): CondNode {
  return { op: 'and', children: [] }
}

export function parseConditionJson(json: string, features?: { feature_key: string }[]): CondNode {
  const s = json?.trim() || ''
  if (!s) return defaultRoot()
  try {
    const o = JSON.parse(s) as CondNode
    if (!o || typeof o !== 'object' || typeof o.op !== 'string') {
      return defaultRoot()
    }
    return normalizeNode(o, features)
  } catch {
    return defaultRoot()
  }
}

function firstFeatureKey(features?: { feature_key: string }[]): string {
  return features?.[0]?.feature_key || ''
}

function normalizeNode(n: CondNode, features?: { feature_key: string }[]): CondNode {
  const op = String(n.op || '').toLowerCase().trim()
  const defaultFeat = firstFeatureKey(features)
  if (op === 'feat') {
    return {
      op: 'feat',
      feature: n.feature || defaultFeat,
      kind: (n.kind || 'true').toLowerCase()
    }
  }
  if (op === 'cmp') {
    return {
      op: 'cmp',
      feature: n.feature || defaultFeat,
      kind: (n.kind || 'gt').toLowerCase(),
      value: typeof n.value === 'number' ? n.value : Number(n.value) || 0
    }
  }
  if (op === 'not') {
    const ch = Array.isArray(n.children) ? n.children.map(c => normalizeNode(c, features)) : []
    return { op: 'not', children: ch.length ? [ch[0]!] : [defaultCmpLeaf(features)] }
  }
  if (op === 'and' || op === 'or') {
    const ch = Array.isArray(n.children) ? n.children.map(c => normalizeNode(c, features)) : []
    return { op, children: ch }
  }
  return defaultRoot()
}

function defaultCmpLeaf(features?: { feature_key: string }[]): CondNode {
  return { op: 'cmp', feature: firstFeatureKey(features), kind: 'gt', value: 0 }
}

export function stringifyCondition(node: CondNode, pretty = false): string {
  return pretty ? JSON.stringify(node, null, 2) : JSON.stringify(node)
}

/** 提交前校验；null 表示通过 */
export function validateConditionTree(n: CondNode | null | undefined): string | null {
  if (!n || !n.op) return '条件无效'
  const op = n.op.toLowerCase()
  if (op === 'and' || op === 'or') {
    const ch = n.children || []
    if (ch.length === 0) return '请至少添加一条子条件，或使用「高级」编辑 JSON'
    for (let i = 0; i < ch.length; i++) {
      const err = validateConditionTree(ch[i])
      if (err) return `子条件 ${i + 1}：${err}`
    }
    return null
  }
  if (op === 'not') {
    const ch = n.children || []
    if (ch.length !== 1) return '「非」必须包含恰好一条子条件'
    return validateConditionTree(ch[0])
  }
  if (op === 'feat') {
    if (!String(n.feature || '').trim()) return '请选择特征（特征为真）'
    return null
  }
  if (op === 'cmp') {
    if (!String(n.feature || '').trim()) return '请选择特征（比较）'
    const k = String(n.kind || '').toLowerCase()
    if (!CMP_KINDS.some((x) => x.value === k)) return '请选择比较方式'
    if (Number.isNaN(Number(n.value))) return '比较值必须是数字'
    return null
  }
  return `未知运算类型：${op}`
}
