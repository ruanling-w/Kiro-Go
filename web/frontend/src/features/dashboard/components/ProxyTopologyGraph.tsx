// ProxyTopologyGraph — interactive canvas: provider nodes → router.
// HTML nodes (with brand logos) over an SVG edge layer. Pan: drag background.
// Zoom: scroll or +/−/fit buttons. Layout auto-centers and fills the card.
import { useRef, useState, useCallback, useMemo, useLayoutEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { AccountListItem } from '@/types/account'
import type { StatusSnapshot } from '@/types/common'
import { bucketOf, providerMeta } from '@/config/providers'
import { ProviderIcon } from '@/components/shared/ProviderIcon'
import { formatCompact } from '@/lib/format'
import { cn } from '@/lib/utils'

interface Props {
  accounts: AccountListItem[]
  stats?: StatusSnapshot
  className?: string
}

const NODE_W = 180
const NODE_H = 56
const ROUTER_SIZE = 84
const RADIUS = 190

// Brand hex per provider bucket (for SVG edges + accents). Kept in sync with
// the Tailwind tints in ModelBrand's PROVIDER_STYLE so topology matches the
// log-table chips (grok = zinc, kiro/claude = violet, …).
const PROVIDER_HEX: Record<string, string> = {
  kiro: '#8b5cf6',
  antigravity: '#0ea5e9',
  grok: '#71717a',
  codex: '#10b981',
  remotekiro: '#14b8a6',
}

function providerHex(p: string) {
  return PROVIDER_HEX[p] ?? '#94a3b8'
}

// Group enabled accounts into provider buckets.
function groupByProvider(accounts: AccountListItem[]) {
  const map = new Map<string, number>()
  for (const a of accounts) {
    if (!a.enabled) continue
    const key = bucketOf(a.provider)
    map.set(key, (map.get(key) ?? 0) + 1)
  }
  return Array.from(map.entries()).map(([provider, count]) => ({ provider, count }))
}

// Radial offsets around center (0,0). Single node sits directly above.
function radialOffsets(n: number, r: number) {
  if (n === 0) return []
  if (n === 1) return [{ x: 0, y: -r }]
  return Array.from({ length: n }, (_, i) => {
    const angle = (2 * Math.PI * i) / n - Math.PI / 2
    return { x: r * Math.cos(angle), y: r * Math.sin(angle) }
  })
}

const clampScale = (s: number) => Math.min(2, Math.max(0.3, s))

export function ProxyTopologyGraph({ accounts, stats, className }: Props) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement>(null)
  const [size, setSize] = useState({ w: 800, h: 460 })
  const [transform, setTransform] = useState({ x: 0, y: 0, scale: 1 })
  const dragging = useRef<{ startX: number; startY: number; tx: number; ty: number } | null>(null)

  useLayoutEffect(() => {
    const el = containerRef.current
    if (!el) return
    const update = () => setSize({ w: el.clientWidth, h: el.clientHeight })
    update()
    const ro = new ResizeObserver(update)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  const providers = useMemo(() => groupByProvider(accounts), [accounts])
  const offsets = useMemo(() => radialOffsets(providers.length, RADIUS), [providers.length])

  const cx = size.w / 2
  const cy = size.h / 2

  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    setTransform((tr) => {
      const factor = e.deltaY < 0 ? 1.1 : 0.9
      return { ...tr, scale: clampScale(tr.scale * factor) }
    })
  }, [])

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if ((e.target as Element).closest('.topo-node')) return
      dragging.current = { startX: e.clientX, startY: e.clientY, tx: transform.x, ty: transform.y }
    },
    [transform],
  )

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (!dragging.current) return
    const dx = e.clientX - dragging.current.startX
    const dy = e.clientY - dragging.current.startY
    setTransform((tr) => ({ ...tr, x: dragging.current!.tx + dx, y: dragging.current!.ty + dy }))
  }, [])

  const handleMouseUp = useCallback(() => {
    dragging.current = null
  }, [])

  const fitView = useCallback(() => setTransform({ x: 0, y: 0, scale: 1 }), [])
  const zoomIn = useCallback(() => setTransform((tr) => ({ ...tr, scale: clampScale(tr.scale * 1.2) })), [])
  const zoomOut = useCallback(() => setTransform((tr) => ({ ...tr, scale: clampScale(tr.scale / 1.2) })), [])

  return (
    <div
      ref={containerRef}
      className={cn('relative h-full min-h-[460px] w-full overflow-hidden rounded-lg select-none', className)}
      style={{ background: 'var(--card)', cursor: dragging.current ? 'grabbing' : 'grab' }}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
      onWheel={handleWheel}
    >
      {/* dot grid background (fixed, doesn't pan) */}
      <svg className="pointer-events-none absolute inset-0 h-full w-full" style={{ opacity: 0.18 }}>
        <defs>
          <pattern id="topo-dots" x="0" y="0" width="26" height="26" patternUnits="userSpaceOnUse">
            <circle cx="1.5" cy="1.5" r="1.2" fill="currentColor" className="text-muted-foreground" />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#topo-dots)" />
      </svg>

      {/* pan/zoom wrapper — holds both edges and nodes so they move together */}
      <div
        className="absolute inset-0"
        style={{
          transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})`,
          transformOrigin: 'center',
        }}
      >
        {/* edge layer */}
        <svg className="pointer-events-none absolute inset-0 h-full w-full" width={size.w} height={size.h}>
          {providers.map((p, i) => {
            const off = offsets[i]
            if (!off) return null
            const color = providerHex(p.provider)
            return (
              <line
                key={`edge-${p.provider}`}
                x1={cx + off.x}
                y1={cy + off.y}
                x2={cx}
                y2={cy}
                stroke={color}
                strokeWidth={2}
                strokeDasharray="6 5"
                opacity={0.55}
              />
            )
          })}
        </svg>

        {/* router node */}
        <div
          className="topo-node absolute flex flex-col items-center justify-center rounded-full border-2 text-center"
          style={{
            left: cx,
            top: cy,
            width: ROUTER_SIZE,
            height: ROUTER_SIZE,
            transform: 'translate(-50%, -50%)',
            borderColor: '#f97316',
            background: 'color-mix(in srgb, #f97316 8%, var(--card))',
          }}
        >
          <span className="font-telemetry text-sm font-bold text-orange-500">
            {stats?.totalRequests !== undefined ? formatCompact(stats.totalRequests) : '—'}
          </span>
          <span className="text-[10px] text-muted-foreground">{t('stats.router')}</span>
        </div>

        {/* provider nodes */}
        {providers.map((p, i) => {
          const off = offsets[i]
          if (!off) return null
          const color = providerHex(p.provider)
          const meta = providerMeta(p.provider)
          const label = meta ? t(meta.labelKey) : p.provider
          return (
            <div
              key={p.provider}
              className="topo-node absolute flex items-center gap-2.5 rounded-xl border px-3 shadow-sm"
              style={{
                left: cx + off.x,
                top: cy + off.y,
                width: NODE_W,
                height: NODE_H,
                transform: 'translate(-50%, -50%)',
                borderColor: color,
                background: `color-mix(in srgb, ${color} 7%, var(--card))`,
              }}
            >
              <span
                className="flex size-8 shrink-0 items-center justify-center rounded-lg"
                style={{ background: `color-mix(in srgb, ${color} 14%, transparent)` }}
              >
                <ProviderIcon provider={p.provider} className="size-5" />
              </span>
              <div className="flex min-w-0 flex-col leading-tight">
                <span className="truncate text-sm font-semibold text-foreground">{label}</span>
                <span className="text-[11px] text-muted-foreground">
                  {t('topology.accounts', { 0: p.count })}
                </span>
              </div>
            </div>
          )
        })}

        {providers.length === 0 && (
          <div
            className="absolute text-sm text-muted-foreground"
            style={{ left: cx, top: cy + 60, transform: 'translate(-50%, -50%)' }}
          >
            {t('topology.noProviders')}
          </div>
        )}
      </div>

      {/* zoom controls */}
      <div className="absolute bottom-3 left-3 flex flex-col gap-1">
        {[
          { label: '+', action: zoomIn },
          { label: '−', action: zoomOut },
          { label: '⤢', action: fitView },
        ].map(({ label, action }) => (
          <button
            key={label}
            onClick={action}
            className="flex h-7 w-7 items-center justify-center rounded border bg-card text-xs font-medium shadow-sm hover:bg-muted"
          >
            {label}
          </button>
        ))}
      </div>
    </div>
  )
}
