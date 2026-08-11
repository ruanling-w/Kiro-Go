import { http } from './httpClient'
import type { ChatAttachment, ChatConversation, ChatImageGenerateResponse, ChatMessage, ChatModel, ChatPage, ChatStreamEvent } from '@/types/chat'

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
  conversations: (status: 'active' | 'archived' = 'active', search = '') => {
    const params = new URLSearchParams({ status, limit: '100' })
    if (search.trim()) params.set('search', search.trim())
    return http.get<ChatPage<ChatConversation>>(`/chat/conversations?${params}`)
  },
  messages: (conversationId: string) => http.get<ChatPage<ChatMessage>>(`/chat/conversations/${conversationId}/messages?limit=200`),
  createConversation: (input: ConversationInput) => http.post<ChatConversation>('/chat/conversations', input),
  updateConversation: (id: string, input: ConversationInput) => http.patch<ChatConversation>(`/chat/conversations/${id}`, input),
  deleteConversation: (id: string) => http.delete<void>(`/chat/conversations/${id}`),
  attachments: (conversationId: string) => http.get<{ data: ChatAttachment[] }>(`/chat/conversations/${conversationId}/attachments`).then((result) => result.data),
  uploadAttachments: async (conversationId: string, files: File[]) => {
    const body = new FormData()
    files.forEach((file) => body.append('files', file))
    const response = await fetch(`/admin/api/chat/conversations/${conversationId}/attachments`, {
      method: 'POST', credentials: 'include', headers: { 'X-CSRF-Token': decodeURIComponent(cookie('kiro_csrf')) }, body,
    })
    if (!response.ok) {
      const error = await response.json().catch(() => null) as { message?: string } | null
      throw new Error(error?.message || `Upload failed (${response.status})`)
    }
    return (await response.json() as { data: ChatAttachment[] }).data
  },
  deleteAttachment: (conversationId: string, attachmentId: string) => http.delete<void>(`/chat/conversations/${conversationId}/attachments/${attachmentId}`),
  generateImage: (
    conversationId: string,
    input: { clientRequestId: string; prompt: string; provider?: string; model?: string; size?: string; quality?: string },
    signal?: AbortSignal,
  ) => http.post<ChatImageGenerateResponse>(`/chat/conversations/${conversationId}/images/generate`, input, { signal }),
  generate: async (
    conversationId: string,
    input: { clientRequestId: string; content: string; attachmentIds?: string[] },
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
