import { useQuery } from '@tanstack/react-query'
import { chatService } from '@/services/chat.service'
import { qk } from '@/config/queryKeys'

export function useChatModels() {
  return useQuery({ queryKey: qk.chatModels, queryFn: chatService.models, staleTime: 60_000 })
}

export function useChatConversations(status: 'active' | 'archived' = 'active', search = '') {
  return useQuery({
    queryKey: [...qk.chatConversations, status, search],
    queryFn: () => chatService.conversations(status, search),
  })
}

export function useChatMessages(conversationId: string) {
  return useQuery({
    queryKey: qk.chatMessages(conversationId),
    queryFn: () => chatService.messages(conversationId),
    enabled: Boolean(conversationId),
  })
}
