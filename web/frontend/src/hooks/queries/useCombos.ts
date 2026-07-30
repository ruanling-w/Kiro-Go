import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { getCombo, listCombos } from '@/services/combos.service'

export function useCombos() {
  return useQuery({ queryKey: qk.combos, queryFn: listCombos })
}

export function useCombo(id: string | null) {
  return useQuery({
    queryKey: id ? qk.combo(id) : ['combos', 'none'],
    queryFn: () => getCombo(id as string),
    enabled: !!id,
  })
}
