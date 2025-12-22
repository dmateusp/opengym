import type { StoryDefault, Story } from '@ladle/react'
import { MarkdownRenderer } from './MarkdownRenderer'

export const Default: Story = () => (
  <div className="max-w-2xl mx-auto p-6">
    <MarkdownRenderer
      value={`# Sunday Morning Volleyball\n\n**Indoor volleyball** — all levels welcome!\n\n- Warm-up at 9:45\n- Games start at 10:00\n\nVisit [our group](https://example.com) for more details.`}
    />
  </div>
)

export default {
  title: 'UI/MarkdownRenderer',
} satisfies StoryDefault
