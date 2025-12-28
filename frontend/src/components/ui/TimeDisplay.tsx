import { useState } from 'react'
import { formatDistanceToNow, format } from 'date-fns'
import { Popover, PopoverContent, PopoverAnchor } from '@/components/ui/popover'

interface TimeDisplayProps {
  /** ISO timestamp string */
  timestamp: string
  /** Display format: 'relative' shows "2 days ago", 'friendly' shows "Tomorrow at 3:00 PM" */
  displayFormat?: 'relative' | 'friendly'
  /** Optional prefix text (e.g., "Created") */
  prefix?: string
  /** Optional CSS classes for the wrapper */
  className?: string
}

export function TimeDisplay({ 
  timestamp, 
  displayFormat = 'relative',
  prefix,
  className = ''
}: TimeDisplayProps) {
  const [isHovered, setIsHovered] = useState(false)
  const date = new Date(timestamp)
  const fullTimestamp = date.toLocaleString()
  
  let displayText = ''
  if (displayFormat === 'relative') {
    displayText = formatDistanceToNow(date, { addSuffix: true })
  } else if (displayFormat === 'friendly') {
    displayText = format(date, "EEEE, do 'of' MMMM 'at' h:mma")
  }

  return (
    <span
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <Popover open={isHovered}>
        <PopoverAnchor asChild>
          <span className={`cursor-help underline decoration-dotted ${className}`}>
            {prefix && `${prefix} `}{displayText}
          </span>
        </PopoverAnchor>
        <PopoverContent side="bottom" className="text-gray-800 text-xs w-auto">
          {fullTimestamp}
        </PopoverContent>
      </Popover>
    </span>
  )
}
