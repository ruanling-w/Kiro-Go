// Validated categorical chart palette (dataviz skill reference instance).
// Slot order IS the CVD-safety mechanism — do not reorder or cycle. For
// all-pairs forms (pie/donut/scatter) only the first 3 slots clear the floors;
// past 3, fold into "Other". Light/dark are the same hues stepped per surface.
//
// We resolve light vs dark at call time from the .dark class on <html> so charts
// recolor with the app's 3-state theme toggle.

export const CHART_LIGHT = [
  '#2a78d6', // blue
  '#eb6834', // orange
  '#1baf7a', // aqua
  '#eda100', // yellow
  '#e87ba4', // magenta
  '#008300', // green
  '#4a3aa7', // violet
  '#e34948', // red
] as const

export const CHART_DARK = [
  '#3987e5',
  '#d95926',
  '#199e70',
  '#c98500',
  '#d55181',
  '#008300',
  '#9085e9',
  '#e66767',
] as const

export const STATUS_COLORS = {
  good: '#0ca30c',
  warning: '#fab219',
  serious: '#ec835a',
  critical: '#d03b3b',
} as const

export function isDark(): boolean {
  return typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
}

/** Categorical hue for a slot index (clamped to the 8-slot ramp). */
export function seriesColor(index: number, dark = isDark()): string {
  const ramp = dark ? CHART_DARK : CHART_LIGHT
  return ramp[index % ramp.length]
}

/** Chart chrome tuned to the current surface. */
export function chartChrome(dark = isDark()) {
  return {
    grid: dark ? '#2c2c2a' : '#e1e0d9',
    axis: dark ? '#383835' : '#c3c2b7',
    tick: '#898781',
    text: dark ? '#c3c2b7' : '#52514e',
    surface: dark ? '#1a1a19' : '#fcfcfb',
  }
}

/** The categorical ramp for the current surface. */
export const CHART_SERIES = new Proxy([] as string[], {
  get(_t, prop) {
    if (typeof prop === 'string' && /^\d+$/.test(prop)) return seriesColor(Number(prop))
    return undefined
  },
}) as unknown as string[]

/** Axis/grid/tooltip styling for Recharts, tuned to the current surface. */
export function chartAxis(dark = isDark()) {
  const c = chartChrome(dark)
  return {
    grid: c.grid,
    tick: c.tick,
    tooltip: {
      background: c.surface,
      border: `1px solid ${c.grid}`,
      borderRadius: 8,
      color: c.text,
      fontSize: 12,
    } as React.CSSProperties,
    tooltipLabel: c.text,
  }
}
