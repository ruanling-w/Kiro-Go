export function parseModelRouting(modelString: string): { provider?: string, model: string } {
  const separator = modelString.indexOf('::')
  if (separator > 0 && modelString.indexOf('::', separator + 2) === -1) {
    return { provider: modelString.slice(0, separator), model: modelString.slice(separator + 2) }
  }
  const slashIdx = modelString.indexOf('/')
  if (slashIdx > 0 && modelString.indexOf('/', slashIdx + 1) === -1) {
    return { provider: modelString.slice(0, slashIdx), model: modelString.slice(slashIdx + 1) }
  }
  return { model: modelString }
}

export function formatModelRouting(provider: string, model: string): string {
  return `${provider}::${model}`
}
