import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { cn } from '@/lib/utils'

interface MarkdownRendererProps {
  value: string
  className?: string
}

export function MarkdownRenderer({ value, className }: MarkdownRendererProps) {
  return (
    <div className={cn('text-gray-800 leading-relaxed', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ node, ...props }) => (
            <a className="text-indigo-600 underline" {...props} />
          ),
          h1: ({ node, ...props }) => (
            <h1 className="text-2xl font-bold mt-6 mb-3" {...props} />
          ),
          h2: ({ node, ...props }) => (
            <h2 className="text-xl font-semibold mt-5 mb-2" {...props} />
          ),
          h3: ({ node, ...props }) => (
            <h3 className="text-lg font-semibold mt-4 mb-2" {...props} />
          ),
          p: ({ node, ...props }) => (
            <p className="mb-3" {...props} />
          ),
          ul: ({ node, ...props }) => (
            <ul className="list-disc pl-5 mb-3" {...props} />
          ),
          ol: ({ node, ...props }) => (
            <ol className="list-decimal pl-5 mb-3" {...props} />
          ),
          code: ({ node, inline, ...props }: any) => (
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
        {value}
      </ReactMarkdown>
    </div>
  )
}
