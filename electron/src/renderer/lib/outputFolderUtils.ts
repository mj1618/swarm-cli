/**
 * Utilities for parsing and formatting output folder names
 * Output folders follow pattern: YYYYMMDD-HHMMSS-[hash]
 * Example: 20260213-142305-abc12345
 */

export interface ParsedOutputFolder {
  date: Date
  hash: string
  shortHash: string
}

/**
 * Parse an output folder name into its components
 * @param name Folder name like "20260213-142305-abc12345"
 * @returns Parsed result or null if name doesn't match expected pattern
 */
export function parseOutputFolderName(name: string): ParsedOutputFolder | null {
  const match = name.match(/^(\d{4})(\d{2})(\d{2})-(\d{2})(\d{2})(\d{2})-([a-f0-9]+)$/i)
  if (!match) return null

  const [, year, month, day, hour, minute, second, hash] = match
  const date = new Date(
    parseInt(year, 10),
    parseInt(month, 10) - 1, // months are 0-indexed
    parseInt(day, 10),
    parseInt(hour, 10),
    parseInt(minute, 10),
    parseInt(second, 10)
  )

  // Validate the date is valid
  if (isNaN(date.getTime())) return null

  return {
    date,
    hash,
    shortHash: hash.slice(0, 6),
  }
}

/**
 * Check if a path is inside an outputs directory
 * @param path Full path to check
 * @returns true if path contains /outputs/ segment
 */
export function isInOutputsDirectory(path: string): boolean {
  return path.includes('/outputs/')
}

/**
 * Format a date as a friendly timestamp
 * - Today: "Today 2:43 PM"
 * - Yesterday: "Yesterday 2:43 PM"
 * - This year: "Feb 13, 2:43 PM"
 * - Older: "Feb 13, 2025 2:43 PM"
 */
export function formatOutputTimestamp(date: Date): string {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  const dateOnly = new Date(date.getFullYear(), date.getMonth(), date.getDate())

  const timeStr = date.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  })

  if (dateOnly.getTime() === today.getTime()) {
    return `Today ${timeStr}`
  }

  if (dateOnly.getTime() === yesterday.getTime()) {
    return `Yesterday ${timeStr}`
  }

  const isThisYear = date.getFullYear() === now.getFullYear()

  if (isThisYear) {
    const monthDay = date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    })
    return `${monthDay}, ${timeStr}`
  }

  const fullDate = date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
  return `${fullDate}, ${timeStr}`
}

/**
 * Get a display label for an output folder
 * @param name Folder name
 * @param path Full path (to verify it's in outputs/)
 * @returns Display object or null if not an output folder
 */
export function getOutputFolderDisplay(
  name: string,
  path: string
): { timestamp: string; hash: string } | null {
  if (!isInOutputsDirectory(path)) return null

  const parsed = parseOutputFolderName(name)
  if (!parsed) return null

  return {
    timestamp: formatOutputTimestamp(parsed.date),
    hash: parsed.shortHash,
  }
}
