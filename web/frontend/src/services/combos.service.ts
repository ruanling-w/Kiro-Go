import { http } from './httpClient'
import type { Combo, ComboInput, ComboListResponse, ComboResetResponse } from '@/types/combo'

export function listCombos(): Promise<Combo[]> {
  return http.get<ComboListResponse>('/combos').then((r) => r.data)
}

export function getCombo(id: string): Promise<Combo> {
  return http.get<Combo>(`/combos/${encodeURIComponent(id)}`)
}

export function createCombo(body: ComboInput): Promise<Combo> {
  return http.post<Combo>('/combos', body)
}

export function updateCombo(id: string, body: ComboInput): Promise<Combo> {
  return http.put<Combo>(`/combos/${encodeURIComponent(id)}`, body)
}

export function deleteCombo(id: string, revision: number): Promise<void> {
  return http.delete<void>(`/combos/${encodeURIComponent(id)}`, { data: { revision } })
}

export function resetComboRotation(id: string, revision: number): Promise<ComboResetResponse> {
  return http.post<ComboResetResponse>(`/combos/${encodeURIComponent(id)}/reset-rotation`, { revision })
}
