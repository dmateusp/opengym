import { useState } from 'react'
import { Button } from '@/components/ui/button'
import CreateGameModal from '@/components/games/CreateGameModal'

export default function HomePage() {
  const [isModalOpen, setIsModalOpen] = useState(false)

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      <div className="container mx-auto px-4 py-16">
        {/* Empty State */}
        <div className="flex flex-col items-center justify-center py-20">
          <div className="text-center">
            <h1 className="text-4xl font-bold text-gray-900 mb-4">
              No games yet 🏐
            </h1>
            <p className="text-lg text-gray-600 mb-8">
              You haven't participated in or organized any games yet. Ready to get started?
            </p>
            <Button
              size="lg"
              onClick={() => setIsModalOpen(true)}
              className="bg-indigo-600 hover:bg-indigo-700"
            >
              Organize a Game
            </Button>
          </div>
        </div>
      </div>

      <CreateGameModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </div>
  )
}
