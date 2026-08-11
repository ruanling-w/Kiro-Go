import { describe, expect, it, vi } from 'vitest'
import { parseChatSSE } from './chat.service'

/** @vitest-environment jsdom */

describe('parseChatSSE', () => {
  it('parses chunked named events and multiline data', async () => {
    const encoder = new TextEncoder()
    const response = new Response(new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode('event: response.delta\r\ndata: {"delta":'))
        controller.enqueue(encoder.encode('"hello"}\r\n\r\n'))
        controller.close()
      },
    }))
    const onEvent = vi.fn()

    await parseChatSSE(response, onEvent)

    expect(onEvent).toHaveBeenCalledWith({ event: 'response.delta', data: { delta: 'hello' } })
  })

  it('ignores an incomplete frame at EOF', async () => {
    const response = new Response('event: response.delta\ndata: {"delta":"partial"}')
    const onEvent = vi.fn()

    await parseChatSSE(response, onEvent)

    expect(onEvent).not.toHaveBeenCalled()
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
