import React from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { cn } from '@/lib/utils'

interface MarkdownRendererProps {
  value: string
  className?: string
}

type CodeProps = React.HTMLAttributes<HTMLElement> & { inline?: boolean; node?: unknown }

export function MarkdownRenderer({ value, className }: MarkdownRendererProps) {
  return (
    <div className={cn('text-gray-800 leading-relaxed', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ ...props }: React.HTMLAttributes<HTMLAnchorElement> & { node?: unknown }) => (
            <a className="text-indigo-600 underline" {...props} />
          ),
          h1: ({ ...props }: React.HTMLAttributes<HTMLHeadingElement> & { node?: unknown }) => (
            <h1 className="text-2xl font-bold mt-6 mb-3" {...props} />
          ),
          h2: ({ ...props }: React.HTMLAttributes<HTMLHeadingElement> & { node?: unknown }) => (
            <h2 className="text-xl font-semibold mt-5 mb-2" {...props} />
          ),
          h3: ({ ...props }: React.HTMLAttributes<HTMLHeadingElement> & { node?: unknown }) => (
            <h3 className="text-lg font-semibold mt-4 mb-2" {...props} />
          ),
          p: ({ ...props }: React.HTMLAttributes<HTMLParagraphElement> & { node?: unknown }) => (
            <p className="mb-3" {...props} />
          ),
          ul: ({ ...props }: React.HTMLAttributes<HTMLUListElement> & { node?: unknown }) => (
            <ul className="list-disc pl-5 mb-3" {...props} />
          ),
          ol: ({ ...props }: React.HTMLAttributes<HTMLOListElement> & { node?: unknown }) => (
            <ol className="list-decimal pl-5 mb-3" {...props} />
          ),
          code: ({ inline, ...props }: CodeProps) => (
            <code
              className={cn(
                'rounded bg-gray-100 px-1 py-0.5 text-sm',
                inline ? '' : 'block p-3'
              )}
              {...props}
            />
          ),
        }}
      >
        {value || ''}
      </ReactMarkdown>
    </div>
  )
}
