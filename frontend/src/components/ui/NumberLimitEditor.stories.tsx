import { useState } from "react"
import { NumberLimitEditor } from "./NumberLimitEditor"

export default {
  title: "UI/NumberLimitEditor"
}

export const Waitlist = () => {
  const [value, setValue] = useState<number | undefined>(5)
  const [isEditing, setIsEditing] = useState(true)

  return (
    <div className="p-8 space-y-4">
      <div>
        <p className="text-sm text-gray-600 mb-2">
          Current value: {value === 0 ? "Disabled" : value === -1 ? "Unlimited" : value}
        </p>
        {isEditing ? (
          <NumberLimitEditor
            value={value}
            onSave={(newValue) => {
              setValue(newValue)
              setIsEditing(false)
            }}
            onCancel={() => setIsEditing(false)}
            placeholder="Enter max waitlist size"
          />
        ) : (
          <button
            onClick={() => setIsEditing(true)}
            className="px-4 py-2 bg-blue-500 text-white rounded"
          >
            Edit Waitlist
          </button>
        )}
      </div>
    </div>
  )
}

export const MaxPlayers = () => {
  const [value, setValue] = useState<number | undefined>(12)
  const [isEditing, setIsEditing] = useState(true)

  return (
    <div className="p-8 space-y-4">
      <div>
        <p className="text-sm text-gray-600 mb-2">
          Current value: {value === -1 ? "Unlimited" : value}
        </p>
        {isEditing ? (
          <NumberLimitEditor
            value={value}
            onSave={(newValue) => {
              setValue(newValue)
              setIsEditing(false)
            }}
            onCancel={() => setIsEditing(false)}
            showDisabledOption={false}
            placeholder="Enter max players"
            label={{
              limited: "Set maximum",
              unlimited: "No limit"
            }}
          />
        ) : (
          <button
            onClick={() => setIsEditing(true)}
            className="px-4 py-2 bg-blue-500 text-white rounded"
          >
            Edit Max Players
          </button>
        )}
      </div>
    </div>
  )
}

export const GuestsPerPlayer = () => {
  const [value, setValue] = useState<number | undefined>(0)
  const [isEditing, setIsEditing] = useState(true)

  return (
    <div className="p-8 space-y-4">
      <div>
        <p className="text-sm text-gray-600 mb-2">
          Current value: {value === 0 ? "No guests" : value === -1 ? "Unlimited guests" : `Up to ${value} guest${value === 1 ? '' : 's'}`}
        </p>
        {isEditing ? (
          <NumberLimitEditor
            value={value}
            onSave={(newValue) => {
              setValue(newValue)
              setIsEditing(false)
            }}
            onCancel={() => setIsEditing(false)}
            placeholder="Enter max guests per player"
            label={{
              disabled: "No guests allowed",
              limited: "Limited guests",
              unlimited: "Unlimited guests"
            }}
          />
        ) : (
          <button
            onClick={() => setIsEditing(true)}
            className="px-4 py-2 bg-blue-500 text-white rounded"
          >
            Edit Guests Per Player
          </button>
        )}
      </div>
    </div>
  )
}

export const StartDisabled = () => {
  const [value, setValue] = useState<number | undefined>(0)
  const [isEditing, setIsEditing] = useState(true)

  return (
    <div className="p-8 space-y-4">
      <div>
        <p className="text-sm text-gray-600 mb-2">
          Current value: {value === 0 ? "Disabled" : value === -1 ? "Unlimited" : value}
        </p>
        {isEditing ? (
          <NumberLimitEditor
            value={value}
            onSave={(newValue) => {
              setValue(newValue)
              setIsEditing(false)
            }}
            onCancel={() => setIsEditing(false)}
            placeholder="Enter value"
          />
        ) : (
          <button
            onClick={() => setIsEditing(true)}
            className="px-4 py-2 bg-blue-500 text-white rounded"
          >
            Edit (starts disabled)
          </button>
        )}
      </div>
    </div>
  )
}

export const StartUnlimited = () => {
  const [value, setValue] = useState<number | undefined>(-1)
  const [isEditing, setIsEditing] = useState(true)

  return (
    <div className="p-8 space-y-4">
      <div>
        <p className="text-sm text-gray-600 mb-2">
          Current value: {value === 0 ? "Disabled" : value === -1 ? "Unlimited" : value}
        </p>
        {isEditing ? (
          <NumberLimitEditor
            value={value}
            onSave={(newValue) => {
              setValue(newValue)
              setIsEditing(false)
            }}
            onCancel={() => setIsEditing(false)}
            placeholder="Enter value"
          />
        ) : (
          <button
            onClick={() => setIsEditing(true)}
            className="px-4 py-2 bg-blue-500 text-white rounded"
          >
            Edit (starts unlimited)
          </button>
        )}
      </div>
    </div>
  )
}
