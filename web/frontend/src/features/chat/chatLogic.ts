import type { ChatMessage, ChatStreamEvent } from '@/types/chat'

export const chatUploadMaxFiles = 4
export const chatUploadMaxBytes = 10 * 1024 * 1024
const chatUploadMIMEs = new Set(['image/png', 'image/jpeg', 'image/webp'])

export interface ChatUploadValidation {
  accepted: File[]
  rejected: 'too_many' | 'invalid_image' | null
}

export function validateChatUploads(current: File[], incoming: File[]): ChatUploadValidation {
  if (current.length + incoming.length > chatUploadMaxFiles) {
    return { accepted: [], rejected: 'too_many' }
  }
  if (incoming.some((file) => file.size === 0 || file.size > chatUploadMaxBytes || !chatUploadMIMEs.has(file.type))) {
    return { accepted: [], rejected: 'invalid_image' }
  }
  return { accepted: [...current, ...incoming], rejected: null }
}

export interface ChatStreamState {
  message: ChatMessage
  reasoning: string
  done: boolean
}

export function reduceChatStream(state: ChatStreamState, event: ChatStreamEvent): ChatStreamState {
  switch (event.event) {
    case 'generation.created':
      return { ...state, message: { ...state.message, id: event.data.assistantMessageId, parentMessageId: event.data.userMessageId } }
    case 'response.delta':
      return { ...state, message: { ...state.message, content: state.message.content + event.data.delta } }
    case 'response.reasoning_summary.delta':
      return { ...state, reasoning: state.reasoning + event.data.delta }
    case 'response.completed':
      return { ...state, message: { ...state.message, status: 'complete', provider: event.data.provider, model: event.data.model, ...event.data.usage } }
    case 'response.error':
      return { ...state, message: { ...state.message, status: 'error', errorCode: event.data.code, errorMessage: event.data.message } }
    case 'done':
      return { ...state, done: true }
  }
}
