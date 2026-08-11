export interface ChatConversation {
  id: string
  title: string
  provider: string
  model: string
  mode: 'chat' | 'image'
  status: 'active' | 'archived'
  pinned: boolean
  projectId: string
  createdAt: number
  updatedAt: number
  archivedAt: number | null
}

export interface ChatMessage {
  id: string
  conversationId: string
  parentMessageId: string
  clientRequestId: string
  role: 'user' | 'assistant'
  content: string
  provider: string
  model: string
  status: 'pending' | 'streaming' | 'complete' | 'stopped' | 'error'
  errorCode: string
  errorMessage: string
  requestId: string
  inputTokens: number
  outputTokens: number
  cacheReadTokens: number
  cacheCreationTokens: number
  createdAt: number
  updatedAt: number
  attachments?: ChatAttachment[]
}

export interface ChatAttachment {
  id: string
  conversationId: string
  messageId: string
  kind: 'image_input' | 'image_output'
  name: string
  mimeType: string
  sizeBytes: number
  width: number | null
  height: number | null
  createdAt: number
  contentUrl: string
}

export interface ChatModel {
  id: string
  provider: string
  model: string
  displayName: string
  availability: string
  capabilities: {
    vision: boolean
    imageGeneration: boolean
    reasoning: boolean
    tools: boolean
    web: boolean
  }
}

export interface ChatPage<T> {
  data: T[]
  nextCursor: string
}

export type ChatStreamEvent =
  | { event: 'generation.created'; data: { generationId: string; userMessageId: string; assistantMessageId: string; replayed?: boolean } }
  | { event: 'response.delta' | 'response.reasoning_summary.delta'; data: { delta: string } }
  | { event: 'response.completed'; data: { finishReason: string; provider: string; model: string; usage: { inputTokens: number; outputTokens: number; cacheReadTokens: number; cacheCreationTokens: number } } }
  | { event: 'response.error'; data: { code: string; message: string; retryable: boolean } }
  | { event: 'done'; data: Record<string, never> }
