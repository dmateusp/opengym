import type { Story } from '@ladle/react'
import { ParticipantCountDisplay } from './ParticipantCountDisplay'

export const Default: Story = () => (
  <ParticipantCountDisplay count={5} maxCount={10} label="Going" />
)

export const Going: Story = () => (
  <ParticipantCountDisplay
    count={12}
    maxCount={20}
    label="Going"
    color="primary"
  />
)

export const Waitlist: Story = () => (
  <ParticipantCountDisplay
    count={3}
    maxCount={5}
    label="Waitlist"
    color="accent"
    showDisabled={true}
  />
)

export const WaitlistDisabled: Story = () => (
  <ParticipantCountDisplay
    count={0}
    maxCount={0}
    label="Waitlist"
    color="accent"
    showDisabled={true}
  />
)

export const WaitlistUnlimited: Story = () => (
  <ParticipantCountDisplay
    count={8}
    maxCount={-1}
    label="Waitlist"
    color="accent"
    showDisabled={true}
  />
)

export const NotGoing: Story = () => (
  <ParticipantCountDisplay count={2} label="Not Going" color="gray" />
)

export const NoMax: Story = () => (
  <ParticipantCountDisplay count={15} label="Going" color="primary" />
)

export const Full: Story = () => (
  <ParticipantCountDisplay count={8} maxCount={8} label="Going" color="primary" />
)
