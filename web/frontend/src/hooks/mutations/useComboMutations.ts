import { useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { createCombo, deleteCombo, resetComboRotation, updateCombo } from '@/services/combos.service'
import type { ComboInput } from '@/types/combo'

export function useCreateCombo() {
  const qc = useQueryClient()
  return useMutation({ mutationFn: createCombo, onSuccess: () => void qc.invalidateQueries({ queryKey: qk.combos }) })
}

export function useUpdateCombo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: ComboInput }) => updateCombo(id, body),
    onSuccess: (_, v) => {
      void qc.invalidateQueries({ queryKey: qk.combos })
      void qc.invalidateQueries({ queryKey: qk.combo(v.id) })
    },
    onError: (error, v) => {
      if ('status' in error && error.status === 409) {
        void qc.invalidateQueries({ queryKey: qk.combos })
        void qc.invalidateQueries({ queryKey: qk.combo(v.id) })
      }
    },
  })
}

export function useDeleteCombo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, revision }: { id: string; revision: number }) => deleteCombo(id, revision),
    onSettled: () => void qc.invalidateQueries({ queryKey: qk.combos }),
  })
}

export function useResetComboRotation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, revision }: { id: string; revision: number }) => resetComboRotation(id, revision),
    onError: (error) => { if ('status' in error && error.status === 409) void qc.invalidateQueries({ queryKey: qk.combos }) },
  })
}
