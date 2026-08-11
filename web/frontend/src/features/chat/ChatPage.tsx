import { useEffect, useMemo, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Archive, Bot, Download, ImagePlus, MessageSquarePlus, Pin, PinOff, RotateCcw, Send, Square, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import { useChatConversations, useChatMessages, useChatModels } from '@/hooks/queries/useChat'
import { chatService } from '@/services/chat.service'
import { qk } from '@/config/queryKeys'
import type { ChatMessage, ChatStreamEvent } from '@/types/chat'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { SafeMarkdown } from '@/components/shared/SafeMarkdown'
import { useConfirm } from '@/components/shared/ConfirmDialog'

function requestId() {
  return crypto.randomUUID()
}

export default function ChatPage() {
  const queryClient = useQueryClient()
  const confirm = useConfirm()
  const models = useChatModels()
  const [conversationStatus, setConversationStatus] = useState<'active' | 'archived'>('active')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const conversations = useChatConversations(conversationStatus, debouncedSearch)
  const [activeId, setActiveId] = useState('')
  const messages = useChatMessages(activeId)
  const [selectedModel, setSelectedModel] = useState('')
  const [draft, setDraft] = useState('')
  const [streaming, setStreaming] = useState<ChatMessage | null>(null)
  const [reasoning, setReasoning] = useState('')
  const [pendingImages, setPendingImages] = useState<File[]>([])
  const [imageMode, setImageMode] = useState(false)
  const [uploading, setUploading] = useState(false)
  const controller = useRef<AbortController | null>(null)
  const fileInput = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    const timeout = window.setTimeout(() => setDebouncedSearch(search), 250)
    return () => window.clearTimeout(timeout)
  }, [search])

  useEffect(() => {
    if (!activeId && conversations.data?.data.length) setActiveId(conversations.data.data[0].id)
  }, [activeId, conversations.data])

  useEffect(() => {
    const conversation = conversations.data?.data.find((item) => item.id === activeId)
    if (conversation) setSelectedModel(`${conversation.provider}:${conversation.model}`)
  }, [activeId, conversations.data])

  useEffect(() => () => controller.current?.abort(), [])

  const transcript = useMemo(() => messages.data?.data ?? [], [messages.data])

  async function createConversation() {
    const model = models.data?.find((item) => item.id === selectedModel) ?? models.data?.[0]
    if (!model) return toast.error('No chat model is available')
    const created = await chatService.createConversation({ provider: model.provider, model: model.model })
    await queryClient.invalidateQueries({ queryKey: qk.chatConversations })
    setActiveId(created.id)
    setSelectedModel(model.id)
  }

  async function removeConversation(id: string) {
    const accepted = await confirm({ title: 'Delete conversation?', description: 'Messages and stored images will be permanently deleted.', confirmLabel: 'Delete', destructive: true })
    if (!accepted) return
    await chatService.deleteConversation(id)
    if (activeId === id) setActiveId('')
    await queryClient.invalidateQueries({ queryKey: qk.chatConversations })
  }

  function addImages(files: File[]) {
    const valid = files.filter((file) => ['image/png', 'image/jpeg', 'image/webp'].includes(file.type) && file.size <= 10 * 1024 * 1024)
    if (valid.length !== files.length) toast.error('Use PNG, JPEG, or WebP images up to 10 MiB')
    setPendingImages((current) => [...current, ...valid].slice(0, 4))
  }

  async function send() {
    const content = draft.trim()
    if ((!content && !pendingImages.length) || controller.current) return
    let conversationId = activeId
    const model = models.data?.find((item) => item.id === selectedModel)
    if (!conversationId) {
      if (!model) return toast.error('Select a model first')
      const created = await chatService.createConversation({ provider: model.provider, model: model.model })
      conversationId = created.id
      setActiveId(created.id)
    } else if (model) {
      await chatService.updateConversation(conversationId, { provider: model.provider, model: model.model })
    }

    const abort = new AbortController()
    controller.current = abort
    if (imageMode) {
      setUploading(true)
      setDraft('')
      try {
        await chatService.generateImage(conversationId, {
          clientRequestId: requestId(), prompt: content, provider: model?.provider, model: model?.model,
        }, abort.signal)
      } catch (error) {
        setDraft(content)
        toast.error(error instanceof Error ? error.message : 'Image generation failed')
      } finally {
        controller.current = null
        setUploading(false)
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: qk.chatMessages(conversationId) }),
          queryClient.invalidateQueries({ queryKey: qk.chatConversations }),
        ])
      }
      return
    }
    setUploading(Boolean(pendingImages.length))
    let attachmentIds: string[] = []
    try {
      if (pendingImages.length) {
        const uploaded = await chatService.uploadAttachments(conversationId, pendingImages)
        attachmentIds = uploaded.map((attachment) => attachment.id)
      }
    } catch (error) {
      controller.current = null
      setUploading(false)
      toast.error(error instanceof Error ? error.message : 'Upload failed')
      return
    }
    setUploading(false)
    setDraft('')
    setPendingImages([])
    setReasoning('')
    setStreaming({
      id: requestId(), conversationId, parentMessageId: '', clientRequestId: '', role: 'assistant', content: '',
      provider: model?.provider ?? '', model: model?.model ?? '', status: 'streaming', errorCode: '', errorMessage: '',
      requestId: '', inputTokens: 0, outputTokens: 0, cacheReadTokens: 0, cacheCreationTokens: 0,
      createdAt: Date.now(), updatedAt: Date.now(),
    })
    try {
      await chatService.generate(conversationId, { clientRequestId: requestId(), content, attachmentIds }, abort.signal, (event: ChatStreamEvent) => {
        if (event.event === 'response.delta') setStreaming((current) => current ? { ...current, content: current.content + event.data.delta } : current)
        if (event.event === 'response.reasoning_summary.delta') setReasoning((current) => current + event.data.delta)
        if (event.event === 'response.completed') setStreaming((current) => current ? {
          ...current, status: 'complete', provider: event.data.provider, model: event.data.model, ...event.data.usage,
        } : current)
        if (event.event === 'response.error') setStreaming((current) => current ? { ...current, status: 'error', errorCode: event.data.code, errorMessage: event.data.message } : current)
      })
    } catch (error) {
      if (!abort.signal.aborted) toast.error(error instanceof Error ? error.message : 'Generation failed')
    } finally {
      controller.current = null
      setStreaming(null)
      setReasoning('')
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: qk.chatMessages(conversationId) }),
        queryClient.invalidateQueries({ queryKey: qk.chatConversations }),
      ])
    }
  }

  return (
    <div className="flex h-[calc(100dvh-7rem)] min-h-[32rem] overflow-hidden rounded-xl border bg-card">
      <aside className="hidden w-72 shrink-0 flex-col border-r bg-muted/20 md:flex">
        <div className="space-y-2 p-3">
          <Button className="w-full" onClick={createConversation}><MessageSquarePlus />New chat</Button>
          <div className="grid grid-cols-2 gap-1">
            <Button size="sm" variant={conversationStatus === 'active' ? 'secondary' : 'ghost'} onClick={() => { setConversationStatus('active'); setActiveId('') }}>Active</Button>
            <Button size="sm" variant={conversationStatus === 'archived' ? 'secondary' : 'ghost'} onClick={() => { setConversationStatus('archived'); setActiveId('') }}>Archived</Button>
          </div>
          <input className="h-9 w-full rounded-md border bg-background px-3 text-sm outline-none" placeholder="Search chats…" value={search} onChange={(event) => setSearch(event.target.value)} />
        </div>
        <div className="flex-1 space-y-1 overflow-y-auto px-2 pb-3">
          {(conversations.data?.data ?? []).map((item) => (
            <div key={item.id} className={`group flex items-center rounded-lg ${activeId === item.id ? 'bg-accent' : 'hover:bg-accent/60'}`}>
              <button className="min-w-0 flex-1 truncate px-3 py-2 text-left text-sm" onClick={() => setActiveId(item.id)}>{item.title || item.model}</button>
              <div className="flex items-center pr-1 opacity-0 group-hover:opacity-100">
                <Button variant="ghost" size="icon" className="size-7" aria-label={item.pinned ? 'Unpin conversation' : 'Pin conversation'} onClick={() => chatService.updateConversation(item.id, { pinned: !item.pinned }).then(() => queryClient.invalidateQueries({ queryKey: qk.chatConversations }))}>{item.pinned ? <PinOff className="size-3.5" /> : <Pin className="size-3.5" />}</Button>
                <Button variant="ghost" size="icon" className="size-7" aria-label={item.status === 'active' ? 'Archive conversation' : 'Restore conversation'} onClick={() => chatService.updateConversation(item.id, { status: item.status === 'active' ? 'archived' : 'active' }).then(() => { if (activeId === item.id) setActiveId(''); return queryClient.invalidateQueries({ queryKey: qk.chatConversations }) })}>{item.status === 'active' ? <Archive className="size-3.5" /> : <RotateCcw className="size-3.5" />}</Button>
                <Button variant="ghost" size="icon" className="size-7" onClick={() => removeConversation(item.id)}><Trash2 className="size-3.5" /></Button>
              </div>
            </div>
          ))}
        </div>
      </aside>

      <section className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center justify-between gap-3 border-b px-4 py-3">
          <div className="flex items-center gap-2 font-medium"><Bot className="size-5" />AI Chat</div>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant={imageMode ? 'default' : 'outline'}
              size="sm"
              onClick={() => {
                const next = !imageMode
                setImageMode(next)
                setPendingImages([])
                if (next) {
                  const imageModel = models.data?.find((item) => item.capabilities.imageGeneration)
                  if (imageModel) setSelectedModel(imageModel.id)
                  else toast.error('No image generation model is available')
                }
              }}
            ><ImagePlus className="size-4" />{imageMode ? 'Create image' : 'Chat'}</Button>
            <Select value={selectedModel} onValueChange={setSelectedModel}>
              <SelectTrigger className="w-[min(22rem,60vw)]"><SelectValue placeholder="Select provider and model" /></SelectTrigger>
              <SelectContent>{(models.data ?? []).filter((model) => !imageMode || model.capabilities.imageGeneration).map((model) => <SelectItem key={model.id} value={model.id}>{model.provider} · {model.displayName}</SelectItem>)}</SelectContent>
            </Select>
          </div>
        </header>

        <div className="flex-1 overflow-y-auto px-4 py-6">
          {!transcript.length && !streaming ? (
            <div className="grid h-full place-items-center text-center text-muted-foreground"><div><Bot className="mx-auto mb-3 size-10" /><p className="font-medium text-foreground">How can I help?</p><p className="text-sm">Choose a provider and model, then start a conversation.</p></div></div>
          ) : (
            <div className="mx-auto max-w-3xl space-y-6">
              {transcript.map((message) => <MessageBubble key={message.id} message={message} />)}
              {streaming && <><MessageBubble message={streaming} />{reasoning && <p className="text-xs text-muted-foreground">Thinking: {reasoning}</p>}</>}
            </div>
          )}
        </div>

        <div className="border-t p-3">
          <div className="mx-auto max-w-3xl">
            {pendingImages.length > 0 && <div className="mb-2 flex gap-2 overflow-x-auto">{pendingImages.map((file, index) => <ImagePreview key={`${file.name}-${file.lastModified}-${index}`} file={file} onRemove={() => setPendingImages((current) => current.filter((_, itemIndex) => itemIndex !== index))} />)}</div>}
            <div
              className="flex items-end gap-2 rounded-xl border bg-background p-2 shadow-sm"
              onDragOver={(event) => event.preventDefault()}
              onDrop={(event) => { event.preventDefault(); addImages(Array.from(event.dataTransfer.files)) }}
            >
              <input ref={fileInput} type="file" accept="image/png,image/jpeg,image/webp" multiple className="hidden" onChange={(event) => { addImages(Array.from(event.target.files ?? [])); event.target.value = '' }} />
              {!imageMode && <Button type="button" size="icon" variant="ghost" onClick={() => fileInput.current?.click()} disabled={Boolean(controller.current) || pendingImages.length >= 4} aria-label="Attach images"><ImagePlus className="size-4" /></Button>}
              <textarea className="max-h-48 min-h-11 flex-1 resize-none bg-transparent px-2 py-2 text-sm outline-none" rows={1} placeholder={imageMode ? 'Describe the image to create…' : 'Message AI…'} value={draft} onChange={(event) => setDraft(event.target.value)} onPaste={(event) => { if (!imageMode) { const files = Array.from(event.clipboardData.files); if (files.length) addImages(files) } }} onKeyDown={(event) => { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); void send() } }} disabled={Boolean(controller.current)} />
              {controller.current ? <Button size="icon" variant="destructive" onClick={() => controller.current?.abort()}><Square className="size-4" /></Button> : <Button size="icon" onClick={send} disabled={(!draft.trim() && !pendingImages.length) || uploading}><Send className="size-4" /></Button>}
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}

function ImagePreview({ file, onRemove }: { file: File; onRemove: () => void }) {
  const [url, setURL] = useState('')
  useEffect(() => {
    const objectURL = URL.createObjectURL(file)
    setURL(objectURL)
    return () => URL.revokeObjectURL(objectURL)
  }, [file])
  return <div className="group relative size-20 shrink-0 overflow-hidden rounded-lg border bg-muted">{url && <img src={url} alt={file.name} className="size-full object-cover" />}<Button type="button" size="icon" variant="secondary" className="absolute right-1 top-1 size-6 opacity-90" onClick={onRemove} aria-label={`Remove ${file.name}`}><X className="size-3" /></Button></div>
}

function MessageBubble({ message }: { message: ChatMessage }) {
  const assistant = message.role === 'assistant'
  const content = message.content || (message.status === 'streaming' ? '…' : message.errorMessage)
  return (
    <div className={`flex ${assistant ? 'justify-start' : 'justify-end'}`}>
      <div className={`min-w-0 max-w-[85%] rounded-2xl px-4 py-3 ${assistant ? 'bg-muted' : 'bg-primary text-primary-foreground'}`}>
        {message.attachments?.length ? <div className="mb-3 grid grid-cols-2 gap-2">{message.attachments.map((attachment) => <div key={attachment.id} className="group/image relative"><img src={attachment.contentUrl} alt={attachment.name} className="max-h-96 w-full rounded-lg object-contain" loading="lazy" />{attachment.kind === 'image_output' && <a href={attachment.contentUrl} download={attachment.name} className="absolute right-2 top-2 grid size-8 place-items-center rounded-md bg-background/90 text-foreground opacity-0 shadow-sm transition-opacity group-hover/image:opacity-100" aria-label={`Download ${attachment.name}`}><Download className="size-4" /></a>}</div>)}</div> : null}
        {assistant ? <SafeMarkdown>{content}</SafeMarkdown> : <div className="whitespace-pre-wrap text-sm">{content}</div>}
        {assistant && message.status === 'error' && <div className="mt-2 text-xs text-destructive">{message.errorMessage}</div>}
        {assistant && <details className="mt-2 text-xs opacity-70"><summary className="cursor-pointer">Details</summary><div className="mt-1 space-y-0.5"><div>{message.provider} · {message.model} · {message.status}</div><div>Input {message.inputTokens} · Output {message.outputTokens}</div><div>Cache read {message.cacheReadTokens} · Cache write {message.cacheCreationTokens}</div>{message.requestId && <div className="break-all">Request {message.requestId}</div>}</div></details>}
      </div>
    </div>
  )
}
