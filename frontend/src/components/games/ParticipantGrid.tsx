import { Crown } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar'
import { TimeDisplay } from '@/components/ui/TimeDisplay'

interface User {
  id: string
  email: string
  name?: string
  picture?: string
}

interface ParticipantWithGuests {
  user: User
  guests?: number
  updatedAt?: string
}

interface ParticipantGridProps {
  participants: (User | ParticipantWithGuests)[]
  organizerId?: number
  /** Total number of slots to display. Empty slots are calculated as totalSlots - occupiedSlots */
  totalSlots?: number
  /** Number of slots actually occupied (including guests). If not provided, defaults to participants.length */
  occupiedSlots?: number
  icon?: LucideIcon
  size?: 'sm' | 'md' | 'lg'
  opacity?: number
  emptySlotLabel?: string
}

const sizeConfig = {
  sm: {
    avatar: 'w-12 h-12',
    crown: 'h-2.5 w-2.5',
    crownPadding: 'p-1',
    text: 'text-[10px] max-w-12',
    emptyIcon: 'text-xs',
  },
  md: {
    avatar: 'w-16 h-16',
    crown: 'h-3 w-3',
    crownPadding: 'p-1.5',
    text: 'text-xs max-w-16',
    emptyIcon: 'text-sm',
  },
  lg: {
    avatar: 'w-20 h-20',
    crown: 'h-4 w-4',
    crownPadding: 'p-2',
    text: 'text-sm max-w-20',
    emptyIcon: 'text-base',
  },
}

export function ParticipantGrid({
  participants,
  organizerId,
  totalSlots,
  occupiedSlots,
  icon: Icon,
  size = 'md',
  opacity = 1
}: ParticipantGridProps) {
  const getInitials = (name?: string, email?: string) => {
    if (name) {
      const parts = name.split(' ').filter((p) => p.length > 0)
      if (parts.length > 0) {
        return parts
          .slice(0, 3)
          .map((p) => p[0].toUpperCase())
          .join('')
      }
      return name.slice(0, 2).toUpperCase()
    }
    if (email) {
      return email.slice(0, 2).toUpperCase()
    }
    return '??'
  }

  const config = sizeConfig[size]
  const containerStyle = opacity < 1 ? { opacity } : undefined
    const { t } = useTranslation();

  return (
    <div className="flex flex-wrap gap-4" style={containerStyle}>
      {participants.map((item) => {
        const participant = 'user' in item ? item.user : item
        const guestCount = 'guests' in item ? item.guests : undefined
        const updatedAt = 'updatedAt' in item ? item.updatedAt : undefined
        const isOrganizerParticipating =
          organizerId && participant.id === String(organizerId)
        return (
          <div
            key={participant.id}
            className="flex flex-col items-center gap-2"
          >
            <div className={`relative ${config.avatar}`}>
              <Avatar
                className={`${config.avatar} ring-4 ring-success/30 shadow-md`}
              >
                <AvatarImage
                  src={participant.picture}
                  alt={participant.name || participant.email}
                />
                <AvatarFallback className="bg-primary text-white font-bold">
                  {getInitials(participant.name, participant.email)}
                </AvatarFallback>
              </Avatar>
              {isOrganizerParticipating && Icon && (
                <div
                  className={`absolute -bottom-1 -right-1 bg-primary rounded-full ${config.crownPadding} shadow-lg border-2 border-white`}
                >
                  <Icon className={`${config.crown} text-white`} />
                </div>
              )}
              {isOrganizerParticipating && !Icon && (
                <div
                  className={`absolute -bottom-1 -right-1 bg-primary rounded-full ${config.crownPadding} shadow-lg border-2 border-white`}
                >
                  <Crown className={`${config.crown} text-white`} />
                </div>
              )}
              {guestCount !== undefined && guestCount > 0 && (
                <div
                  className={`absolute -top-1 -right-1 bg-secondary rounded-full px-2 py-1 shadow-lg border-2 border-white flex items-center justify-center`}
                  title={`${guestCount} guest${guestCount !== 1 ? 's' : ''}`}
                >
                  <span className="text-white font-bold text-xs leading-none">
                    +{guestCount}
                  </span>
                </div>
              )}
            </div>
            <div className="flex flex-col items-center gap-0.5">
              <span
                className={`${config.text} text-center text-gray-700 font-medium truncate`}
              >
                {participant.name || participant.email}
              </span>
              {updatedAt && (
                <TimeDisplay 
                  timestamp={updatedAt} 
                  displayFormat="relative"
                  className="text-[10px] text-gray-500"
                />
              )}
            </div>
          </div>
        )
      })}
      {/* Empty slots */}
      {totalSlots &&
        totalSlots > 0 &&
        Array.from({
          length: Math.max(0, totalSlots - (occupiedSlots ?? participants.length)),
        }).map((_, i) => (
          <div
            key={`empty-${i}`}
            className="flex flex-col items-center gap-2"
          >
            <div
              className={`${config.avatar} rounded-full border-4 border-dashed border-gray-300 bg-gray-50 flex items-center justify-center`}
            >
              <span className={`text-gray-400 ${config.emptyIcon}`}>?</span>
            </div>
            <span
              className={`${config.text} text-center text-gray-400 font-medium truncate`}
            >
              {t('game.emptySlot')}
            </span>
          </div>
        ))}
    </div>
  )
}
