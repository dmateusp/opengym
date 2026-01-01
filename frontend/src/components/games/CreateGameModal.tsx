import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { API_BASE_URL, redirectToLogin } from '@/lib/api'
import { Loader2 } from 'lucide-react'

interface CreateGameModalProps {
  isOpen: boolean
  onClose: () => void
}

export default function CreateGameModal({ isOpen, onClose }: CreateGameModalProps) {
  const navigate = useNavigate()
  const [gameName, setGameName] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!gameName.trim()) {
      setError('Game name is required')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const response = await fetch(`${API_BASE_URL}/api/games`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          name: gameName.trim(),
        }),
      })

      if (!response.ok) {
        if (response.status === 401) {
          redirectToLogin()
          return
        }
        const errorText = await response.text()
        throw new Error(errorText || 'Failed to create game')
      }

      const game = await response.json()
      
      // Reset and close modal
      setGameName('')
      setError(null)
      onClose()
      
      // Navigate to game detail page
      navigate(`/games/${game.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setIsLoading(false)
    }
  }

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      // Reset state when closing
      setGameName('')
      setError(null)
      onClose()
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px] rounded-2xl">
        <DialogHeader className="space-y-2">
          <DialogTitle className="text-2xl">Organize a Game</DialogTitle>
          <DialogDescription className="text-base">
            What are you playing?
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="space-y-2">
            <label htmlFor="game-name" className="text-sm font-semibold text-gray-700">
              Game Name
            </label>
            <Input
              id="game-name"
              placeholder="e.g., Sunday Morning Volleyball"
              value={gameName}
              onChange={(e) => setGameName(e.target.value)}
              disabled={isLoading}
              autoFocus
              className="rounded-xl border-2 focus:border-primary focus:ring-0 text-base"
            />
            <p className="text-xs text-gray-500">Keep it simple and fun</p>
          </div>

          {error && (
            <div className="text-sm text-white bg-red-500 p-4 rounded-xl">
              {error}
            </div>
          )}

          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isLoading}
              className="rounded-full"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isLoading || !gameName.trim()}
              className="bg-accent rounded-full"
            >
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isLoading ? 'Creating...' : 'Create'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
