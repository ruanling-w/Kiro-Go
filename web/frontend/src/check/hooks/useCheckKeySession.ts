// Holds the API key the user is checking on /check. Seeded from sessionStorage
// so a refresh in the same tab restores the dashboard; clear() drops both.
import { useCallback, useState } from 'react'
import { clearCheckKey, readCheckKey, writeCheckKey } from '../lib/checkKeyStorage'

export function useCheckKeySession() {
  const [key, setKeyState] = useState<string | null>(() => readCheckKey())

  const setKey = useCallback((next: string) => {
    const trimmed = next.trim()
    writeCheckKey(trimmed)
    setKeyState(trimmed)
  }, [])

  const clear = useCallback(() => {
    clearCheckKey()
    setKeyState(null)
  }, [])

  return { key, setKey, clear }
}
