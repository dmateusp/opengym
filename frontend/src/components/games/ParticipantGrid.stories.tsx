import type { Story } from '@ladle/react'
import { ParticipantGrid } from './ParticipantGrid'
import { Crown, Clock } from 'lucide-react'

const mockParticipants = [
  {
    user: {
      id: '1',
      email: 'alice@example.com',
      name: 'Alice Johnson',
      picture: 'https://placecats.com/150/150?image=1',
    },
    updatedAt: new Date(Date.now() - 1000 * 60 * 5).toISOString(), // 5 minutes ago
  },
  {
    user: {
      id: '2',
      email: 'bob@example.com',
      name: 'Bob Smith',
      picture: 'https://placecats.com/150/150?image=2',
    },
    updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(), // 2 hours ago
  },
  {
    user: {
      id: '3',
      email: 'charlie@example.com',
      name: 'Charlie Brown',
    },
    updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString(), // 1 day ago
  },
  {
    user: {
      id: '4',
      email: 'diana@example.com',
      name: 'Diana Prince',
      picture: 'https://placecats.com/150/150?image=4',
    },
    updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 3).toISOString(), // 3 days ago
  },
]

export const Default: Story = () => (
  <ParticipantGrid participants={mockParticipants} totalSlots={8} />
)

export const WithOrganizer: Story = () => (
  <ParticipantGrid
    participants={mockParticipants}
    organizerId={1}
    totalSlots={8}
    icon={Crown}
  />
)

export const SmallSize: Story = () => (
  <ParticipantGrid
    participants={mockParticipants}
    organizerId={2}
    totalSlots={6}
    size="sm"
    icon={Crown}
  />
)

export const LargeSize: Story = () => (
  <ParticipantGrid
    participants={mockParticipants.slice(0, 2)}
    organizerId={1}
    totalSlots={4}
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
        totalSlots={8}
        icon={Crown}
        opacity={1}
      />
    </div>
    <div>
      <h3 className="mb-4 font-semibold">Lower Opacity (Waitlist)</h3>
      <ParticipantGrid
        participants={mockParticipants.slice(0, 2)}
        organizerId={1}
        totalSlots={4}
        icon={Clock}
        size="sm"
        opacity={0.7}
      />
    </div>
  </div>
)

export const NoMaxCount: Story = () => (
  <ParticipantGrid participants={mockParticipants} organizerId={1} icon={Crown} />
)

export const Empty: Story = () => <ParticipantGrid participants={[]} totalSlots={8} />

export const Full: Story = () => (
  <ParticipantGrid
    participants={[
      ...mockParticipants,
      {
        user: {
          id: '5',
          email: 'eve@example.com',
          name: 'Eve Wilson',
          picture: 'https://placecats.com/150/150?image=5',
        },
        updatedAt: new Date(Date.now() - 1000 * 60 * 30).toISOString(), // 30 minutes ago
      },
      {
        user: {
          id: '6',
          email: 'frank@example.com',
          name: 'Frank Castle',
        },
        updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 5).toISOString(), // 5 hours ago
      },
      {
        user: {
          id: '7',
          email: 'grace@example.com',
          name: 'Grace Lee',
          picture: 'https://placecats.com/150/150?image=7',
        },
        updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 2).toISOString(), // 2 days ago
      },
      {
        user: {
          id: '8',
          email: 'hank@example.com',
          name: 'Hank Pym',
          picture: 'https://placecats.com/150/150?image=8',
        },
        updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 7).toISOString(), // 7 days ago
      },
    ]}
    organizerId={3}
    totalSlots={8}
    icon={Crown}
  />
)

export const WithGuests: Story = () => (
  <ParticipantGrid
    participants={[
      mockParticipants[0],
      {
        user: mockParticipants[1].user,
        guests: 2,
      },
      {
        user: mockParticipants[2].user,
        guests: 1,
      },
    ]}
    totalSlots={5}
    organizerId={1}
    icon={Crown}
  />
)
