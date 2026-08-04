// Credential parsing for the JSON/batch import, ported from importCredentials()
// and parseLineCredentials() in web/js/auth-modals.js. Kept out of the component
// so the shapes stay testable and the form stays presentational.

/** One normalized account ready to POST. */
export interface CredentialItem {
  refreshToken: string
  accessToken?: string
  clientId?: string
  clientSecret?: string
  region?: string
  authMethod?: string
  provider?: string
}

export interface ParseResult {
  items: CredentialItem[]
  /** Line-format rows that did not have enough fields to use. */
  skipped: number
  /** True when the input was not JSON at all and the line parser ran instead. */
  usedLineFormat: boolean
}

function str(v: unknown): string | undefined {
  return typeof v === 'string' && v.trim() !== '' ? v.trim() : undefined
}

/** Pull one account out of either a raw object or an export-bundle entry. */
function fromObject(raw: unknown): CredentialItem | null {
  if (typeof raw !== 'object' || raw === null) return null
  const a = raw as Record<string, unknown>
  const c = (typeof a.credentials === 'object' && a.credentials !== null
    ? a.credentials
    : {}) as Record<string, unknown>

  const refreshToken = str(c.refreshToken) ?? str(a.refreshToken)
  if (!refreshToken) return null

  // The export bundle prettifies authMethod as "IdC"; normalize back.
  const authMethod = str(c.authMethod) ?? str(a.authMethod)
  return {
    refreshToken,
    accessToken: str(c.accessToken) ?? str(a.accessToken),
    clientId: str(c.clientId) ?? str(a.clientId),
    clientSecret: str(c.clientSecret) ?? str(a.clientSecret),
    region: str(c.region) ?? str(a.region),
    authMethod: authMethod?.toLowerCase(),
    // `idp` is what /export emits for the login channel.
    provider: str(c.provider) ?? str(a.provider) ?? str(a.idp),
  }
}

/**
 * Line format: email----password----refreshToken----clientId----clientSecret.
 * The separator is auto-detected (----, tab, or runs of spaces), matching the
 * legacy parser. Rows with fewer than 3 fields (no refreshToken) are skipped.
 */
export function parseLineCredentials(raw: string): { items: CredentialItem[]; skipped: number } {
  const items: CredentialItem[] = []
  let skipped = 0

  for (const line of raw.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed) continue

    let parts: string[]
    if (trimmed.includes('----')) parts = trimmed.split('----')
    else if (trimmed.includes('\t')) parts = trimmed.split('\t')
    else parts = trimmed.split(/\s+/)
    parts = parts.map((p) => p.trim())

    const refreshToken = parts[2]
    if (!refreshToken) {
      skipped++
      continue
    }
    const clientId = parts[3] || undefined
    const clientSecret = parts[4] || undefined
    items.push({
      refreshToken,
      clientId,
      clientSecret,
      // clientId+secret present ⇒ IdC; otherwise a social refresh token.
      authMethod: clientId && clientSecret ? 'idc' : 'social',
    })
  }
  return { items, skipped }
}

/**
 * Accepts any of: the Kiro Account Manager export bundle ({accounts:[...]}), a
 * bare array of accounts, a single account object, or the `----` line format.
 * Throws when the input is neither valid JSON nor parseable as lines.
 */
export function parseCredentialsInput(raw: string): ParseResult {
  const text = raw.trim()
  if (!text) throw new Error('empty')

  try {
    const json: unknown = JSON.parse(text)
    let candidates: unknown[]
    if (typeof json === 'object' && json !== null && Array.isArray((json as { accounts?: unknown }).accounts)) {
      candidates = (json as { accounts: unknown[] }).accounts
    } else {
      candidates = Array.isArray(json) ? json : [json]
    }
    const items: CredentialItem[] = []
    let skipped = 0
    for (const c of candidates) {
      const item = fromObject(c)
      if (item) items.push(item)
      else skipped++
    }
    return { items, skipped, usedLineFormat: false }
  } catch (e) {
    if (e instanceof Error && e.message === 'empty') throw e
    // Not JSON — fall back to the line format.
    const { items, skipped } = parseLineCredentials(text)
    if (items.length === 0 && skipped === 0) throw new Error('unparseable')
    return { items, skipped, usedLineFormat: true }
  }
}

/** A fully-defaulted account, matching what the server expects on the wire. */
export interface NormalizedCredential {
  refreshToken: string
  accessToken: string
  clientId: string
  clientSecret: string
  authMethod: string
  provider: string
  region: string
}

/**
 * Fill in the authMethod/provider defaults the legacy importer applied before
 * posting, so the server sees the same payload shape it always did.
 */
export function normalizeForPost(item: CredentialItem): NormalizedCredential {
  let authMethod = item.authMethod ?? ''
  if (item.clientId && item.clientSecret) authMethod = 'idc'
  else if (authMethod === 'external_idp' || authMethod === 'entra' || authMethod === 'azuread') {
    authMethod = 'external_idp'
  } else if (!authMethod || authMethod === 'social') authMethod = 'social'
  else authMethod = authMethod === 'idc' ? 'idc' : 'social'

  let provider = item.provider ?? ''
  if (!provider && authMethod === 'social') provider = 'Google'
  if (!provider && authMethod === 'idc') provider = 'BuilderId'

  return {
    refreshToken: item.refreshToken,
    accessToken: item.accessToken ?? '',
    clientId: item.clientId ?? '',
    clientSecret: item.clientSecret ?? '',
    authMethod,
    provider,
    region: item.region || 'us-east-1',
  }
}
