// Email masking for privacy mode. Ported from the legacy maskEmail (web/js/core.js):
// keeps first 2 chars of each significant segment, masks the rest with ***.

export function maskEmail(email: string): string {
  if (!email || !email.includes('@')) return email
  const [local, domain] = email.split('@')
  const maskedLocal = local.length <= 2 ? local : local.slice(0, 2) + '***'
  const parts = domain.split('.')
  if (parts.length >= 2) {
    const tld = parts[parts.length - 1]
    const sld = parts[parts.length - 2]
    const maskedSld = sld.length <= 2 ? sld : sld.slice(0, 2) + '***'
    const subs = parts.slice(0, -2).map((s) => (s.length <= 2 ? s : s.slice(0, 2) + '***'))
    return maskedLocal + '@' + [...subs, maskedSld, tld].join('.')
  }
  return maskedLocal + '@' + domain
}

/** Falls back to a truncated id when there's no email. */
export function displayEmail(email: string, id: string, privacy: boolean): string {
  const raw = email || (id ? id.slice(0, 12) + '...' : '-')
  return privacy ? maskEmail(raw) : raw
}
