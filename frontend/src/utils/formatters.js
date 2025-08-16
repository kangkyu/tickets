import { format, parseISO } from 'date-fns'

export const formatSatsToUSD = (sats, btcPrice = 50000) => {
  const btc = sats / 100000000
  return (btc * btcPrice).toFixed(2)
}

// Robust date parser that handles various ISO formats
const parseDate = (dateString) => {
  if (!dateString) return null
  
  try {
    // First try parseISO from date-fns
    if (typeof dateString === 'string') {
      return parseISO(dateString)
    }
    return dateString
  } catch (error) {
    try {
      // Fallback to native Date constructor
      return new Date(dateString)
    } catch (fallbackError) {
      console.error('Failed to parse date:', dateString, fallbackError)
      return null
    }
  }
}

export const formatEventDate = (dateString) => {
  try {
    const date = parseDate(dateString)
    if (!date || isNaN(date.getTime())) {
      console.error('Invalid date object:', date, 'from string:', dateString)
      return 'Invalid date'
    }
    return format(date, 'PPP p')
  } catch (error) {
    console.error('Error formatting date:', dateString, error)
    return 'Invalid date'
  }
}

export const formatEventDateShort = (dateString) => {
  try {
    const date = parseDate(dateString)
    if (!date || isNaN(date.getTime())) {
      return 'Invalid date'
    }
    return format(date, 'MMM d, yyyy')
  } catch (error) {
    console.error('Error formatting short date:', dateString, error)
    return 'Invalid date'
  }
}

export const truncateText = (text, maxLength = 100) => {
  if (!text) return ''
  return text.length > maxLength ? text.slice(0, maxLength) + '...' : text
}

export const formatPrice = (sats) => {
  if (sats < 1000) return `${sats} sats`
  if (sats < 1000000) return `${(sats / 1000).toFixed(1)}k sats`
  return `${(sats / 1000000).toFixed(2)}M sats`
}

export const formatTimeAgo = (dateString) => {
  try {
    const date = typeof dateString === 'string' ? parseISO(dateString) : dateString
    const now = new Date()
    const diffInSeconds = Math.floor((now - date) / 1000)
    
    if (diffInSeconds < 60) return 'Just now'
    if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`
    if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`
    return `${Math.floor(diffInSeconds / 86400)}d ago`
  } catch (error) {
    return 'Unknown time'
  }
}
