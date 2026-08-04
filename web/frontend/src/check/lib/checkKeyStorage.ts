// Tab-scoped storage for the raw API key used by the public /check page.
// sessionStorage only — closing the tab clears it; never write to localStorage.
const STORAGE_KEY = 'kiro_check_api_key'

export function readCheckKey(): string | null {
  try {
    const v = sessionStorage.getItem(STORAGE_KEY)
    return v && v.trim() ? v.trim() : null
  } catch {
    return null
  }
}

export function writeCheckKey(key: string): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, key.trim())
  } catch {
    // private mode / quota — ignore; in-memory state still works for the session
  }
}

export function clearCheckKey(): void {
  try {
    sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
}
