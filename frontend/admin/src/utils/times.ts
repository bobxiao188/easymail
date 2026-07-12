import dayjs from 'dayjs'

export function formatDateTime(input?: string | number | Date | null): string {
  if (!input) return ''
  const d = dayjs(input)
  if (!d.isValid()) return ''
  return d.format('YYYY-MM-DD HH:mm:ss')
}

export function formatDate(input?: string | number | Date | null): string {
  if (!input) return ''
  const d = dayjs(input)
  if (!d.isValid()) return ''
  return d.format('YYYY-MM-DD')
}

export function formatTime(input?: string | number | Date | null): string {
  if (!input) return ''
  const d = dayjs(input)
  if (!d.isValid()) return ''
  return d.format('HH:mm:ss')
}

export function isZeroTime(input?: string | number | Date | null): boolean {
  if (!input) return true
  const d = dayjs(input)
  if (!d.isValid()) return true
  // 兼容后端可能返回的 Go 零值时间（0001-01-01）以及早期时区偏移导致的奇怪时间串（如 0001-01-01 08:05:43）
  return d.year() <= 1 || d.isSame(dayjs('0000-00-00 00:00:00')) || d.isSame(dayjs('0001-01-01T00:00:00Z'))
}