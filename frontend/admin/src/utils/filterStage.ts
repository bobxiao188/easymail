/** Pipeline ordinal 0..5 (matches backend feature.Stage). */
const STAGE_MIN = 0
const STAGE_MAX = 6

/**
 * Human-readable pipeline stage for Filter Logs / admin UI.
 * Returns i18n name plus numeric code, e.g. "HEADERS (4)".
 */
export function formatPipelineStageLabel(
  t: (key: string, values?: Record<string, unknown>) => string,
  stage: number | null | undefined
): string {
  if (stage === null || stage === undefined || Number.isNaN(Number(stage))) {
    return '—'
  }
  const n = Math.floor(Number(stage))
  if (n < STAGE_MIN || n > STAGE_MAX) {
    return t('scannerStage.unknown', { n: String(stage) })
  }
  const name = t(`scannerStage.names.${n}`)
  return `${name} (${n})`
}
