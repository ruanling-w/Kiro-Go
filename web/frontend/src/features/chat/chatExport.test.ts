import { describe, expect, it } from 'vitest'
import type { ChatConversation, ChatMessage } from '@/types/chat'
import { chatExportJSON, chatExportMarkdown } from './chatExport'

const conversation: ChatConversation = {
  id: 'conversation', title: 'Export test', provider: 'kiro', model: 'claude', mode: 'chat',
  status: 'active', pinned: false, projectId: '', createdAt: 1, updatedAt: 2, archivedAt: null,
}

const messages: ChatMessage[] = [{
  id: 'user', conversationId: conversation.id, parentMessageId: '', clientRequestId: 'request', role: 'user',
  content: 'Describe this', provider: 'kiro', model: 'claude', status: 'complete', errorCode: '', errorMessage: '',
  requestId: '', inputTokens: 0, outputTokens: 0, cacheReadTokens: 0, cacheCreationTokens: 0, createdAt: 1, updatedAt: 1,
  attachments: [{ id: 'image', conversationId: conversation.id, messageId: 'user', kind: 'image_input', name: 'photo.png', mimeType: 'image/png', sizeBytes: 12, width: 2, height: 3, createdAt: 1, contentUrl: '/admin/api/chat/assets/photo' }],
}, {
  id: 'assistant', conversationId: conversation.id, parentMessageId: 'user', clientRequestId: '', role: 'assistant',
  content: 'A photo.', provider: 'kiro', model: 'claude', status: 'complete', errorCode: '', errorMessage: '',
  requestId: 'upstream', inputTokens: 4, outputTokens: 2, cacheReadTokens: 1, cacheCreationTokens: 0, createdAt: 2, updatedAt: 2,
}]

describe('chat export formatters', () => {
  it('exports structured JSON without embedding image bytes', () => {
    const output = chatExportJSON(conversation, messages, '2026-08-12T00:00:00.000Z')
    const parsed = JSON.parse(output)
    expect(parsed.exportedAt).toBe('2026-08-12T00:00:00.000Z')
    expect(parsed.messages[0].attachments[0].contentUrl).toBe('/admin/api/chat/assets/photo')
    expect(output).not.toContain('base64')
  })

  it('exports Markdown roles, model details, and attachment references', () => {
    const output = chatExportMarkdown(conversation, messages, '2026-08-12T00:00:00.000Z')
    expect(output).toContain('# Export test')
    expect(output).toContain('## User\n\nDescribe this')
    expect(output).toContain('[photo.png](/admin/api/chat/assets/photo)')
    expect(output).toContain('## Assistant\n\nA photo.')
    expect(output).toContain('Model: kiro:claude')
  })
})
