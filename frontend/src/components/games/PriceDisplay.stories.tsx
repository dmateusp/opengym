import type { StoryDefault, Story } from '@ladle/react'
import { PriceDisplay } from './PriceDisplay'

export const Default: Story = () => (
  <div className="max-w-md mx-auto p-6 space-y-4">
    <PriceDisplay totalPriceCents={1500} currentPlayers={6} maxPlayers={12} />
    <PriceDisplay totalPriceCents={2000} currentPlayers={3} maxPlayers={8} />
    <PriceDisplay totalPriceCents={3000} currentPlayers={0} maxPlayers={0} />
    <PriceDisplay totalPriceCents={undefined} currentPlayers={4} maxPlayers={12} />
  </div>
)

export default {
  title: 'Games/PriceDisplay',
} satisfies StoryDefault
