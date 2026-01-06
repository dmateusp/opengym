import type { Story } from '@ladle/react'
import { ParticipantGrid } from './ParticipantGrid'
import { Crown, Clock } from 'lucide-react'

const mockParticipants = [
  {
    id: '1',
    email: 'alice@example.com',
    name: 'Alice Johnson',
    picture: 'https://placecats.com/150/150?image=1',
  },
  {
    id: '2',
    email: 'bob@example.com',
    name: 'Bob Smith',
    picture: 'https://placecats.com/150/150?image=2',
  },
  {
    id: '3',
    email: 'charlie@example.com',
    name: 'Charlie Brown',
  },
  {
    id: '4',
    email: 'diana@example.com',
    name: 'Diana Prince',
    picture: 'https://placecats.com/150/150?image=4',
  },
]

export const Default: Story = () => (
  <ParticipantGrid participants={mockParticipants} maxCount={8} />
)

export const WithOrganizer: Story = () => (
  <ParticipantGrid
    participants={mockParticipants}
    organizerId={1}
    maxCount={8}
    icon={Crown}
  />
)

export const SmallSize: Story = () => (
  <ParticipantGrid
    participants={mockParticipants}
    organizerId={2}
    maxCount={6}
    size="sm"
    icon={Crown}
  />
)

export const LargeSize: Story = () => (
  <ParticipantGrid
    participants={mockParticipants.slice(0, 2)}
    organizerId={1}
    maxCount={4}
    size="lg"
    icon={Crown}
  />
)

export const WithOpacity: Story = () => (
  <div className="space-y-8">
    <div>
      <h3 className="mb-4 font-semibold">Full Opacity (Participants)</h3>
      <ParticipantGrid
        participants={mockParticipants}
        organizerId={1}
        maxCount={8}
        icon={Crown}
        opacity={1}
      />
    </div>
    <div>
      <h3 className="mb-4 font-semibold">Lower Opacity (Waitlist)</h3>
      <ParticipantGrid
        participants={mockParticipants.slice(0, 2)}
        organizerId={1}
        maxCount={4}
        icon={Clock}
        size="sm"
        opacity={0.7}
        emptySlotLabel="Available"
      />
    </div>
  </div>
)

export const NoMaxCount: Story = () => (
  <ParticipantGrid participants={mockParticipants} organizerId={1} icon={Crown} />
)

export const Empty: Story = () => <ParticipantGrid participants={[]} maxCount={8} />

export const Full: Story = () => (
  <ParticipantGrid
    participants={[
      ...mockParticipants,
      {
        id: '5',
        email: 'eve@example.com',
        name: 'Eve Wilson',
        picture: 'https://placecats.com/150/150?image=5',
      },
      {
        id: '6',
        email: 'frank@example.com',
        name: 'Frank Castle',
      },
      {
        id: '7',
        email: 'grace@example.com',
        name: 'Grace Lee',
        picture: 'https://placecats.com/150/150?image=7',
      },
      {
        id: '8',
        email: 'hank@example.com',
        name: 'Hank Pym',
        picture: 'https://placecats.com/150/150?image=8',
      },
    ]}
    organizerId={3}
    maxCount={8}
    icon={Crown}
  />
)

export const CustomEmptyLabel: Story = () => (
  <ParticipantGrid
    participants={mockParticipants.slice(0, 2)}
    maxCount={5}
    emptySlotLabel="Join"
  />
)
