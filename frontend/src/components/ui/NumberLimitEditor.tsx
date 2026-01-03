import { Input } from "./input"

export type LimitMode = 'disabled' | 'limited' | 'unlimited'

export interface NumberLimitEditorProps {
  value: number | undefined
  onSave: (value: number | undefined) => void
  onCancel: () => void
  showDisabledOption?: boolean
  placeholder?: string
  label?: {
    disabled?: string
    limited?: string
    unlimited?: string
  }
}

export function NumberLimitEditor({
  value,
  onSave,
  onCancel,
  showDisabledOption = true,
  placeholder = "Enter max size",
  label = {
    disabled: "Disabled",
    limited: "Limited size",
    unlimited: "Unlimited"
  }
}: NumberLimitEditorProps) {
  const [mode, setMode] = React.useState<LimitMode>(() => {
    if (value === 0) return 'disabled'
    if (value === -1) return 'unlimited'
    return 'limited'
  })
  
  const [limitValue, setLimitValue] = React.useState<string>(() => {
    if (typeof value === 'number' && value > 0) {
      return String(value)
    }
    return ''
  })

  // If disabled option is hidden, ensure we start with limited mode
  React.useEffect(() => {
    if (!showDisabledOption && mode === 'disabled') {
      setMode('limited')
    }
  }, [showDisabledOption, mode])

  const handleBlur = (e: React.FocusEvent<HTMLDivElement>) => {
    // Only trigger blur if focus is leaving the container
    if (!e.currentTarget.contains(e.relatedTarget as Node)) {
      let valueToSave: number | undefined
      
      if (mode === 'disabled') {
        valueToSave = 0
      } else if (mode === 'unlimited') {
        valueToSave = -1
      } else {
        const num = Number(limitValue)
        valueToSave = isNaN(num) || num < 1 ? undefined : num
      }
      
      if (valueToSave !== undefined) {
        onSave(valueToSave)
      } else {
        onCancel()
      }
    }
  }

  return (
    <div
      className="space-y-2 p-3 border-2 border-primary rounded-xl bg-white"
      onBlur={handleBlur}
    >
      {showDisabledOption && (
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            name="limitMode"
            value="disabled"
            checked={mode === 'disabled'}
            onChange={(e) => setMode(e.target.value as LimitMode)}
            className="w-4 h-4 text-primary"
          />
          <span className="text-sm font-medium">{label.disabled}</span>
        </label>
      )}
      
      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="radio"
          name="limitMode"
          value="limited"
          checked={mode === 'limited'}
          onChange={(e) => setMode(e.target.value as LimitMode)}
          className="w-4 h-4 text-primary"
        />
        <span className="text-sm font-medium">{label.limited}</span>
      </label>
      
      {mode === 'limited' && (
        <div className="pl-6">
          <Input
            type="number"
            value={limitValue}
            onChange={(e) => setLimitValue(e.target.value)}
            placeholder={placeholder}
            className="w-full"
            min="1"
            autoFocus
          />
        </div>
      )}
      
      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="radio"
          name="limitMode"
          value="unlimited"
          checked={mode === 'unlimited'}
          onChange={(e) => setMode(e.target.value as LimitMode)}
          className="w-4 h-4 text-primary"
        />
        <span className="text-sm font-medium">{label.unlimited}</span>
      </label>
    </div>
  )
}

// Need to import React for useState
import React from "react"
