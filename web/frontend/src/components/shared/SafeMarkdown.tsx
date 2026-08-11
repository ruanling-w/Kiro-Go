import type { ComponentPropsWithoutRef } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CodeBlock } from '@/components/shared/CodeBlock'
import { cn } from '@/lib/utils'

interface SafeMarkdownProps {
  children: string
  className?: string
}

function safeURL(url: string) {
  const value = url.trim()
  if (/^(https?:|mailto:)/i.test(value)) return value
  if (/^(\/|#)/.test(value)) return value
  return ''
}

export function SafeMarkdown({ children, className }: SafeMarkdownProps) {
  return (
    <div className={cn('min-w-0 break-words text-sm leading-7', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        urlTransform={safeURL}
        skipHtml
        components={{
          a: ({ href, children: label, ...props }) => {
            const external = /^https?:/i.test(href ?? '')
            return <a {...props} href={href} target={external ? '_blank' : undefined} rel={external ? 'noopener noreferrer' : undefined} className="text-primary underline underline-offset-4">{label}</a>
          },
          p: ({ children: content }) => <p className="my-3 first:mt-0 last:mb-0">{content}</p>,
          ul: ({ children: content }) => <ul className="my-3 list-disc space-y-1 pl-6">{content}</ul>,
          ol: ({ children: content }) => <ol className="my-3 list-decimal space-y-1 pl-6">{content}</ol>,
          blockquote: ({ children: content }) => <blockquote className="my-3 border-l-2 border-border pl-4 text-muted-foreground">{content}</blockquote>,
          h1: ({ children: content }) => <h1 className="mb-3 mt-5 text-xl font-semibold first:mt-0">{content}</h1>,
          h2: ({ children: content }) => <h2 className="mb-3 mt-5 text-lg font-semibold first:mt-0">{content}</h2>,
          h3: ({ children: content }) => <h3 className="mb-2 mt-4 font-semibold first:mt-0">{content}</h3>,
          table: ({ children: content }) => <div className="my-4 overflow-x-auto"><table className="w-full border-collapse text-left text-sm">{content}</table></div>,
          th: ({ children: content }) => <th className="border bg-muted/60 px-3 py-2 font-medium">{content}</th>,
          td: ({ children: content }) => <td className="border px-3 py-2 align-top">{content}</td>,
          code: ({ className: codeClass, children: content, ...props }: ComponentPropsWithoutRef<'code'>) => {
            const match = /language-([^\s]+)/.exec(codeClass ?? '')
            const code = String(content).replace(/\n$/, '')
            if (match || code.includes('\n')) return <CodeBlock code={code} lang={match?.[1]} className="my-4" />
            return <code {...props} className="rounded bg-muted px-1.5 py-0.5 font-mono text-[0.9em]">{content}</code>
          },
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
