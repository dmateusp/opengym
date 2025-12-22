import { useState } from 'react'
import { formatDistanceToNow, format } from 'date-fns'
import { enUS, pt } from 'date-fns/locale'
import { useTranslation } from 'react-i18next'
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
  const { i18n, t } = useTranslation()
  const [isHovered, setIsHovered] = useState(false)
  const date = new Date(timestamp)
  
  // Map i18next language codes to date-fns locales
  const dateFnsLocale = i18n.language === 'pt-PT' ? pt : enUS
  const localeCode = i18n.language === 'pt-PT' ? 'pt-PT' : 'en-US'
  
  const fullTimestamp = date.toLocaleString(localeCode)
  
  let displayText = ''
  if (displayFormat === 'relative') {
    displayText = formatDistanceToNow(date, { addSuffix: true, locale: dateFnsLocale })
  } else if (displayFormat === 'friendly') {
    const formatString = t('common.dateFormatFriendly')
    displayText = format(date, formatString, { locale: dateFnsLocale })
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
