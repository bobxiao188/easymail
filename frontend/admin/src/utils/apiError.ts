import type { AxiosError } from 'axios'

/** Backend JSON envelope from Gin handlers */
type ApiLike = { message?: string; code?: number }

/**
 * Prefer the API `message` field (often localized by the server); otherwise use fallback.
 * Accepts an Axios error, or a plain `{ message?: string }` response body.
 */
export function messageFromApiError(err: unknown, fallback: string): string {
  if (typeof err === 'object' && err !== null && 'message' in err) {
    const m = (err as { message?: unknown }).message
    if (typeof m === 'string') {
      const s = m.trim()
      if (s.length > 0) {
        return s
      }
    }
  }
  const ax = err as AxiosError<ApiLike>
  const data = ax.response?.data
  const m = data?.message
  if (typeof m === 'string') {
    const s = m.trim()
    if (s.length > 0) {
      return s
    }
  }
  return fallback
}
