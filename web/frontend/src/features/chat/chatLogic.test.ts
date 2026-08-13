import { describe, expect, it } from 'vitest'
import type { ChatMessage } from '@/types/chat'
import { reduceChatStream, validateChatUploads } from './chatLogic'

function file(name: string, type: string, size = 1) {
  return new File([new Uint8Array(size)], name, { type })
}

const message: ChatMessage = {
  id: 'local', conversationId: 'conversation', parentMessageId: '', clientRequestId: '', role: 'assistant', content: '',
  provider: '', model: '', status: 'streaming', errorCode: '', errorMessage: '', requestId: '', inputTokens: 0,
  outputTokens: 0, cacheReadTokens: 0, cacheCreationTokens: 0, createdAt: 1, updatedAt: 1,
}

describe('validateChatUploads', () => {
  it('accepts supported images within the four-file limit', () => {
    const result = validateChatUploads([file('one.png', 'image/png')], [file('two.jpg', 'image/jpeg'), file('three.webp', 'image/webp')])
    expect(result.rejected).toBeNull()
    expect(result.accepted).toHaveLength(3)
  })

  it.each([
    [file('empty.png', 'image/png', 0), 'empty'],
    [file('bad.gif', 'image/gif'), 'unsupported'],
    [file('large.png', 'image/png', 10 * 1024 * 1024 + 1), 'oversized'],
  ])('rejects %s images', (invalid) => {
    expect(validateChatUploads([], [invalid]).rejected).toBe('invalid_image')
  })

  it('rejects the complete batch instead of silently dropping overflow', () => {
    const current = Array.from({ length: 3 }, (_, index) => file(`${index}.png`, 'image/png'))
    const result = validateChatUploads(current, [file('four.png', 'image/png'), file('five.png', 'image/png')])
    expect(result).toEqual({ accepted: [], rejected: 'too_many' })
  })
})

describe('reduceChatStream', () => {
  it('reduces ids, reasoning, deltas, usage, and done', () => {
    let state = { message, reasoning: '', done: false }
    state = reduceChatStream(state, { event: 'generation.created', data: { generationId: 'g', userMessageId: 'user', assistantMessageId: 'assistant' } })
    state = reduceChatStream(state, { event: 'response.reasoning_summary.delta', data: { delta: 'think' } })
    state = reduceChatStream(state, { event: 'response.delta', data: { delta: 'hello' } })
    state = reduceChatStream(state, { event: 'response.completed', data: { finishReason: 'stop', provider: 'kiro', model: 'claude', usage: { inputTokens: 2, outputTokens: 1, cacheReadTokens: 1, cacheCreationTokens: 0 } } })
    state = reduceChatStream(state, { event: 'done', data: {} })
    expect(state.message).toMatchObject({ id: 'assistant', parentMessageId: 'user', content: 'hello', status: 'complete', provider: 'kiro', inputTokens: 2 })
    expect(state.reasoning).toBe('think')
    expect(state.done).toBe(true)
  })

  it('persists terminal stream errors', () => {
    const state = reduceChatStream({ message, reasoning: '', done: false }, { event: 'response.error', data: { code: 'provider_error', message: 'failed', retryable: true } })
    expect(state.message).toMatchObject({ status: 'error', errorCode: 'provider_error', errorMessage: 'failed' })
  })
})
