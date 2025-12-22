import type { StoryDefault } from '@ladle/react'
import { TimeDisplay } from './TimeDisplay'

export default {
  title: 'TimeDisplay',
} satisfies StoryDefault<typeof TimeDisplay>

// Helper to create timestamps
function getTimestamp(daysAgo: number): string {
  const date = new Date()
  date.setDate(date.getDate() - daysAgo)
  return date.toISOString()
}

function getFutureTimestamp(daysAhead: number, hours: number = 14, minutes: number = 30): string {
  const date = new Date()
  date.setDate(date.getDate() + daysAhead)
  date.setHours(hours, minutes, 0, 0)
  return date.toISOString()
}

// Relative Format Stories
export function RelativeFormat() {
  return (
    <div className="space-y-4 p-6">
      <div>
        <p className="text-sm text-gray-600 mb-2">Created 2 days ago:</p>
        <TimeDisplay 
          timestamp={getTimestamp(2)} 
          displayFormat="relative"
          prefix="Created"
        />
      </div>
      <div>
        <p className="text-sm text-gray-600 mb-2">Updated 1 hour ago:</p>
        <TimeDisplay 
          timestamp={getTimestamp(0)} 
          displayFormat="relative"
          prefix="Updated"
        />
      </div>
      <div>
        <p className="text-sm text-gray-600 mb-2">Just now (same date):</p>
        <TimeDisplay 
          timestamp={new Date().toISOString()} 
          displayFormat="relative"
        />
      </div>
    </div>
  )
}

// Friendly Format Stories
export function FriendlyFormat() {
  return (
    <div className="space-y-4 p-6">
      <div>
        <p className="text-sm text-gray-600 mb-2">Future date (Monday, 29th of December at 2:30pm):</p>
        <TimeDisplay 
          timestamp={getFutureTimestamp(1, 14, 30)} 
          displayFormat="friendly"
        />
      </div>
      <div>
        <p className="text-sm text-gray-600 mb-2">Further in future (Wednesday, 31st of December at 6:00pm):</p>
        <TimeDisplay 
          timestamp={getFutureTimestamp(3, 18, 0)} 
          displayFormat="friendly"
        />
      </div>
      <div>
        <p className="text-sm text-gray-600 mb-2">Next month (Friday, 9th of January at 11:00am):</p>
        <TimeDisplay 
          timestamp={getFutureTimestamp(12, 11, 0)} 
          displayFormat="friendly"
        />
      </div>
    </div>
  )
}

// Without Prefix
export function WithoutPrefix() {
  return (
    <div className="space-y-4 p-6">
      <div>
        <p className="text-sm text-gray-600 mb-2">Relative without prefix:</p>
        <TimeDisplay 
          timestamp={getTimestamp(3)} 
          displayFormat="relative"
        />
      </div>
      <div>
        <p className="text-sm text-gray-600 mb-2">Friendly without prefix:</p>
        <TimeDisplay 
          timestamp={getFutureTimestamp(2, 19, 0)} 
          displayFormat="friendly"
        />
      </div>
    </div>
  )
}

// Styled Variants
export function StyledVariants() {
  return (
    <div className="space-y-6 p-6">
      <div className="bg-indigo-50 p-4 rounded-lg">
        <p className="text-sm text-gray-600 mb-2">Header style (white/light):</p>
        <div className="bg-indigo-600 text-white p-4 rounded">
          <TimeDisplay 
            timestamp={getTimestamp(5)} 
            displayFormat="relative"
            prefix="Created"
            className="text-white/80 decoration-white/40"
          />
        </div>
      </div>
      
      <div className="bg-gray-50 p-4 rounded-lg">
        <p className="text-sm text-gray-600 mb-2">Card style (gray text):</p>
        <div className="bg-white p-4 rounded border border-gray-200">
          <TimeDisplay 
            timestamp={getFutureTimestamp(1, 15, 30)} 
            displayFormat="friendly"
            className="text-gray-700 decoration-gray-400"
          />
        </div>
      </div>

      <div className="bg-gray-50 p-4 rounded-lg">
        <p className="text-sm text-gray-600 mb-2">Small muted style (for tables):</p>
        <div className="text-xs text-gray-500">
          <TimeDisplay 
            timestamp={getTimestamp(1)} 
            displayFormat="relative"
            className="text-gray-500 decoration-gray-400"
          />
        </div>
      </div>
    </div>
  )
}

// Use Case: Game Detail Page Scenarios
export function GameDetailPageScenarios() {
  return (
    <div className="space-y-8 p-6">
      <div className="bg-gradient-to-r from-indigo-600 to-purple-600 text-white p-6 rounded-lg">
        <h3 className="text-lg font-semibold mb-4">Game Header</h3>
        <div className="flex items-center gap-2 text-sm text-white/80">
          <span>📅</span>
          <TimeDisplay 
            timestamp={getTimestamp(7)} 
            displayFormat="relative"
            prefix="Created"
            className="text-white/80 decoration-white/40"
          />
        </div>
      </div>

      <div className="bg-gray-50 rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-4">Start Time Field</h3>
        <div className="bg-white p-4 rounded border border-gray-200">
          <TimeDisplay 
            timestamp={getFutureTimestamp(2, 19, 30)} 
            displayFormat="friendly"
            className="text-gray-700 decoration-gray-400"
          />
        </div>
      </div>
    </div>
  )
}

// Use Case: Game Listing Scenarios
export function GameListingScenarios() {
  return (
    <div className="space-y-4 p-6">
      <table className="w-full text-sm border-collapse">
        <thead>
          <tr className="border-b">
            <th className="text-left py-2 px-3 font-semibold text-gray-700">Name</th>
            <th className="text-left py-2 px-3 font-semibold text-gray-700">When</th>
            <th className="text-right py-2 px-3 font-semibold text-gray-700">Updated</th>
          </tr>
        </thead>
        <tbody>
          <tr className="border-b hover:bg-gray-50">
            <td className="py-3 px-3">Football Match</td>
            <td className="py-3 px-3 text-xs">
              <TimeDisplay 
                timestamp={getFutureTimestamp(1, 14, 0)} 
                displayFormat="friendly"
                className="text-gray-600 decoration-gray-400"
              />
            </td>
            <td className="py-3 px-3 text-right text-xs">
              <TimeDisplay 
                timestamp={getTimestamp(0)} 
                displayFormat="relative"
                className="text-gray-500 decoration-gray-400"
              />
            </td>
          </tr>
          <tr className="border-b hover:bg-gray-50">
            <td className="py-3 px-3">Board Game Night</td>
            <td className="py-3 px-3 text-xs">
              <TimeDisplay 
                timestamp={getFutureTimestamp(5, 19, 30)} 
                displayFormat="friendly"
                className="text-gray-600 decoration-gray-400"
              />
            </td>
            <td className="py-3 px-3 text-right text-xs">
              <TimeDisplay 
                timestamp={getTimestamp(2)} 
                displayFormat="relative"
                className="text-gray-500 decoration-gray-400"
              />
            </td>
          </tr>
        </tbody>
      </table>
      <p className="text-xs text-gray-500 mt-4">Hover over any timestamp to see the full date and time</p>
    </div>
  )
}
