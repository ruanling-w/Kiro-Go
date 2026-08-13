export function parseModelRouting(modelString: string): { provider?: string, model: string } {
  const separator = modelString.indexOf('::')
  if (separator > 0 && modelString.indexOf('::', separator + 2) === -1) {
    return { provider: modelString.slice(0, separator), model: modelString.slice(separator + 2) }
  }
  return { model: modelString }
}

export function formatModelRouting(provider: string, model: string): string {
  return `${provider}::${model}`
}
