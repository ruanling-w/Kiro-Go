// LogsTerminal — renders request logs as ANSI-colored terminal lines via xterm.
// Each log is one line: time · status · endpoint · model · provider · account ·
// IP · apiKey · tokens · duration. Status/errorType drives the color. The theme
// syncs with the app's light/dark. Search uses @xterm/addon-search.
import { useEffect, useRef, useState } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { SearchAddon } from '@xterm/addon-search'
import '@xterm/xterm/css/xterm.css'
import type { RequestLog } from '@/types/log'
import { formatClockSeconds, formatNumber, formatDuration } from '@/lib/format'
import { isDark } from '@/lib/chartColors'

const RESET = '\x1b[0m'
const BOLD = '\x1b[1m'

// Truecolor palette per theme. On the light (beige) background we use deep,
// saturated shades so text stays legible instead of the washed-out bright ANSI.
type Palette = {
  green: string; red: string; yellow: string; cyan: string
  magenta: string; blue: string; orange: string; teal: string; gray: string
}

function fg(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `\x1b[38;2;${r};${g};${b}m`
}

const LIGHT_PALETTE: Palette = {
  green: fg('#137a4b'),
  red: fg('#c0362c'),
  yellow: fg('#9a6b00'),
  cyan: fg('#0e7490'),
  magenta: fg('#9333ea'),
  blue: fg('#1d4ed8'),
  orange: fg('#b45309'),
  teal: fg('#0f766e'),
  gray: fg('#6b7280'),
}

const DARK_PALETTE: Palette = {
  green: fg('#4ade80'),
  red: fg('#f87171'),
  yellow: fg('#fbbf24'),
  cyan: fg('#22d3ee'),
  magenta: fg('#e879f9'),
  blue: fg('#60a5fa'),
  orange: fg('#fb923c'),
  teal: fg('#2dd4bf'),
  gray: fg('#9ca3af'),
}

function lightTheme() {
  return { background: '#f0efe9', foreground: '#1a1a19', cursor: '#1a1a19' }
}
function darkTheme() {
  return { background: '#1a1a19', foreground: '#e5e5e5', cursor: '#e5e5e5' }
}

function field(p: Palette, label: string, value: string, color: string): string {
  return `${p.gray}${label}:${RESET}${color}${value}${RESET}`
}

function formatLine(log: RequestLog, keyNames: Map<string, string>, p: Palette): string {
  const ok = log.status === 'success'
  const statusColor = ok ? p.green : p.red
  const statusText = ok ? 'OK ' : 'ERR'
  const keyName = log.apiKeyId ? keyNames.get(log.apiKeyId) : ''
  const parts = [
    `${p.gray}${formatClockSeconds(log.time)}${RESET}`,
    `${BOLD}${statusColor}${statusText}${RESET}`,
    field(p, 'model', log.model || '-', p.blue),
    field(p, 'api', log.endpoint || '-', p.cyan),
    log.provider ? field(p, 'via', log.provider, p.magenta) : '',
    keyName ? field(p, 'key', keyName, p.yellow) : '',
    log.accountId ? field(p, 'acct', log.accountId, p.teal) : '',
    field(p, 'tok', formatNumber(log.tokens), p.green),
    log.credits ? field(p, 'cr', formatNumber(log.credits), p.orange) : '',
    field(p, 'took', formatDuration(log.duration), p.magenta),
    log.clientIp ? field(p, 'ip', log.clientIp, p.cyan) : '',
    log.errorType ? field(p, 'err', log.errorType, p.red) : '',
    !ok && log.error ? `${p.red}${log.error}${RESET}` : '',
  ].filter(Boolean)
  return parts.join(`${p.gray} · ${RESET}`)
}

interface Props {
  logs: RequestLog[]
  keyNames: Map<string, string>
  searchTerm?: string
}

export function LogsTerminal({ logs, keyNames, searchTerm }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const searchRef = useRef<SearchAddon | null>(null)
  const [dark, setDark] = useState(isDark())

  useEffect(() => {
    if (!containerRef.current) return
    const term = new Terminal({
      convertEol: true,
      fontSize: 14,
      lineHeight: 1.5,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      theme: isDark() ? darkTheme() : lightTheme(),
      scrollback: 5000,
      disableStdin: true,
    })
    const fit = new FitAddon()
    const search = new SearchAddon()
    term.loadAddon(fit)
    term.loadAddon(search)
    term.open(containerRef.current)
    fit.fit()
    termRef.current = term
    fitRef.current = fit
    searchRef.current = search

    const onResize = () => fit.fit()
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('resize', onResize)
      term.dispose()
      termRef.current = null
    }
  }, [])

  // Re-theme when the app theme changes.
  useEffect(() => {
    const obs = new MutationObserver(() => {
      const d = isDark()
      setDark(d)
      if (termRef.current) {
        termRef.current.options.theme = d ? darkTheme() : lightTheme()
      }
    })
    obs.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
    return () => obs.disconnect()
  }, [])

  // Rewrite the buffer whenever the logs change.
  useEffect(() => {
    const term = termRef.current
    if (!term) return
    term.clear()
    const p = dark ? DARK_PALETTE : LIGHT_PALETTE
    if (logs.length === 0) {
      term.writeln(`${p.gray}—${RESET}`)
      return
    }
    for (const log of logs) term.writeln(formatLine(log, keyNames, p))
    fitRef.current?.fit()
  }, [logs, keyNames, dark])

  useEffect(() => {
    if (searchTerm && searchRef.current) searchRef.current.findNext(searchTerm)
  }, [searchTerm])

  return <div ref={containerRef} className="h-[78vh] w-full overflow-hidden rounded-lg border bg-card p-5" />
}
