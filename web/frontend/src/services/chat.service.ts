import { http } from './httpClient'
import type { ChatConversation, ChatMessage, ChatModel, ChatPage, ChatStreamEvent } from '@/types/chat'

interface ConversationInput {
  title?: string
  provider?: string
  model?: string
  status?: 'active' | 'archived'
  pinned?: boolean
}

function cookie(name: string) {
  const prefix = `${name}=`
  return document.cookie.split(';').map((value) => value.trim()).find((value) => value.startsWith(prefix))?.slice(prefix.length) ?? ''
}

export async function parseChatSSE(response: Response, onEvent: (event: ChatStreamEvent) => void) {
  if (!response.ok) {
    let message = `Request failed (${response.status})`
    try {
      const body = await response.json() as { message?: string; error?: string }
      message = body.message || body.error || message
    } catch {
      // Keep the HTTP fallback when the body is not JSON.
    }
    throw new Error(message)
  }
  if (!response.body) throw new Error('Streaming response has no body')

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value, { stream: !done }).replace(/\r\n/g, '\n')
    let boundary = buffer.indexOf('\n\n')
    while (boundary >= 0) {
      const frame = buffer.slice(0, boundary)
      buffer = buffer.slice(boundary + 2)
      let event = ''
      const data: string[] = []
      for (const line of frame.split('\n')) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        if (line.startsWith('data:')) data.push(line.slice(5).trimStart())
      }
      if (event && data.length) {
        onEvent({ event, data: JSON.parse(data.join('\n')) } as ChatStreamEvent)
      }
      boundary = buffer.indexOf('\n\n')
    }
    if (done) break
  }
}

export const chatService = {
  models: () => http.get<{ data: ChatModel[] }>('/chat/models').then((result) => result.data),
  conversations: () => http.get<ChatPage<ChatConversation>>('/chat/conversations?status=active&limit=100'),
  messages: (conversationId: string) => http.get<ChatPage<ChatMessage>>(`/chat/conversations/${conversationId}/messages?limit=200`),
  createConversation: (input: ConversationInput) => http.post<ChatConversation>('/chat/conversations', input),
  updateConversation: (id: string, input: ConversationInput) => http.patch<ChatConversation>(`/chat/conversations/${id}`, input),
  deleteConversation: (id: string) => http.delete<void>(`/chat/conversations/${id}`),
  generate: async (
    conversationId: string,
    input: { clientRequestId: string; content: string },
    signal: AbortSignal,
    onEvent: (event: ChatStreamEvent) => void,
  ) => {
    const response = await fetch(`/admin/api/chat/conversations/${conversationId}/generate`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'text/event-stream',
        'Content-Type': 'application/json',
        'X-CSRF-Token': decodeURIComponent(cookie('kiro_csrf')),
      },
      body: JSON.stringify(input),
      signal,
    })
    await parseChatSSE(response, onEvent)
  },
}
