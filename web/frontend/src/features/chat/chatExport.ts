import type { ChatAttachment, ChatConversation, ChatMessage } from '@/types/chat'

export interface ChatExport {
  exportedAt: string
  conversation: ChatConversation
  messages: ChatMessage[]
}

function attachmentLine(attachment: ChatAttachment) {
  const dimensions = attachment.width && attachment.height
    ? ` (${attachment.width}×${attachment.height})`
    : ''
  return `- [${attachment.name}](${attachment.contentUrl}) — ${attachment.mimeType}, ${attachment.sizeBytes} bytes${dimensions}`
}

export function chatExportJSON(conversation: ChatConversation, messages: ChatMessage[], exportedAt = new Date().toISOString()) {
  return JSON.stringify({ exportedAt, conversation, messages } satisfies ChatExport, null, 2)
}

export function chatExportMarkdown(conversation: ChatConversation, messages: ChatMessage[], exportedAt = new Date().toISOString()) {
  const title = conversation.title || `${conversation.provider}:${conversation.model}`
  const sections = messages.map((message) => {
    const heading = message.role === 'user' ? 'User' : 'Assistant'
    const details = message.role === 'assistant'
      ? `\n\n_Status: ${message.status} · Model: ${message.provider}:${message.model}_`
      : ''
    const attachments = message.attachments?.length
      ? `\n\nAttachments:\n${message.attachments.map(attachmentLine).join('\n')}`
      : ''
    return `## ${heading}\n\n${message.content}${attachments}${details}`
  })
  return `# ${title}\n\nExported: ${exportedAt}\n\n${sections.join('\n\n')}`
}

export function downloadChatExport(filename: string, content: string, type: string) {
  const url = URL.createObjectURL(new Blob([content], { type }))
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}
