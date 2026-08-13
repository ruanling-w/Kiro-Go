import { describe, expect, it, vi } from 'vitest'
import { parseChatSSE } from './chat.service'

/** @vitest-environment jsdom */

const completed = 'event: response.completed\ndata: {"finishReason":"stop","provider":"kiro","model":"claude","usage":{"inputTokens":1,"outputTokens":1,"cacheReadTokens":0,"cacheCreationTokens":0}}\n\n'
const done = 'event: done\ndata: {}\n\n'

describe('parseChatSSE', () => {
  it('parses chunked named events and multiline data through completion', async () => {
    const encoder = new TextEncoder()
    const response = new Response(new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('event: response.delta\r\ndata: {"delta":'))
        controller.enqueue(encoder.encode('"hello"}\r\n\r\n'))
        controller.enqueue(encoder.encode(completed + done))
        controller.close()
      },
    }))
    const onEvent = vi.fn()

    await parseChatSSE(response, onEvent)

    expect(onEvent).toHaveBeenCalledWith({ event: 'response.delta', data: { delta: 'hello' } })
    expect(onEvent).toHaveBeenLastCalledWith({ event: 'done', data: {} })
  })

  it('parses the final frame without a trailing blank line', async () => {
    const onEvent = vi.fn()
    const response = new Response(completed + 'event: done\ndata: {}')

    await parseChatSSE(response, onEvent)

    expect(onEvent).toHaveBeenLastCalledWith({ event: 'done', data: {} })
  })

  it('rejects EOF before terminal stream events', async () => {
    const response = new Response('event: response.delta\ndata: {"delta":"partial"}')

    await expect(parseChatSSE(response, vi.fn())).rejects.toThrow('ended before completion')
  })

  it('accepts an error terminal followed by done', async () => {
    const onEvent = vi.fn()
    const response = new Response('event: response.error\ndata: {"code":"provider_error","message":"failed","retryable":true}\n\n' + done)

    await parseChatSSE(response, onEvent)

    expect(onEvent).toHaveBeenNthCalledWith(1, {
      event: 'response.error',
      data: { code: 'provider_error', message: 'failed', retryable: true },
    })
  })

  it('rejects malformed JSON in a complete frame', async () => {
    const response = new Response('event: response.delta\ndata: {invalid}\n\n')

    await expect(parseChatSSE(response, vi.fn())).rejects.toThrow()
  })

  it('throws normalized HTTP errors', async () => {
    await expect(parseChatSSE(new Response(JSON.stringify({ error: 'failed' }), {
      status: 422,
      headers: { 'Content-Type': 'application/json' },
    }), vi.fn())).rejects.toThrow('failed')
  })
})
