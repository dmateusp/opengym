import type { Story } from '@ladle/react'
import UserProfileMenu from './UserProfileMenu'
import type { User } from '@/opengym/client'

const mockUser: User = {
  id: '123',
  email: 'john.doe@example.com',
  name: 'John Doe',
  picture: 'https://api.dicebear.com/7.x/avataaars/svg?seed=John',
  isDemo: true,
}

const mockUserNoPicture: User = {
  id: '456',
  email: 'jane.smith@example.com',
  name: 'Jane Smith',
  isDemo: false,
}

const mockUserNoName: User = {
  id: '789',
  email: 'user@example.com',
  isDemo: true,
}

export const WithPicture: Story = () => (
  <div className="flex justify-end p-8 bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen">
    <UserProfileMenu user={mockUser} />
  </div>
)

export const WithoutPicture: Story = () => (
  <div className="flex justify-end p-8 bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen">
    <UserProfileMenu user={mockUserNoPicture} />
  </div>
)

export const WithoutName: Story = () => (
  <div className="flex justify-end p-8 bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen">
    <UserProfileMenu user={mockUserNoName} />
  </div>
)

export const NullUser: Story = () => (
  <div className="flex justify-end p-8 bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen">
    <UserProfileMenu user={null} />
  </div>
)
