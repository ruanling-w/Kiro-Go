// export.service — POST /export returns a Kiro Account Manager-compatible JSON
// bundle. Timestamps inside are MILLISECONDS (unlike accounts/logs which are
// unix seconds — see plan). Empty ids = export all.
import { http } from './httpClient'

/** The export payload is a passthrough JSON blob; we don't reshape it. */
export type ExportBundle = Record<string, unknown>

export function exportAccounts(ids?: string[]): Promise<ExportBundle> {
  return http.post<ExportBundle>('/export', { ids: ids ?? [] })
}
