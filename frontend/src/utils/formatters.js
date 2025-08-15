import { format, parseISO } from 'date-fns'

export const formatSatsToUSD = (sats, btcPrice = 50000) => {
  const btc = sats / 100000000
  return (btc * btcPrice).toFixed(2)
}

export const formatEventDate = (dateString) => {
  try {
    const date = typeof dateString === 'string' ? parseISO(dateString) : dateString
    return format(date, 'PPP p')
  } catch (error) {
    return 'Invalid date'
  }
}

export const formatEventDateShort = (dateString) => {
  try {
    const date = typeof dateString === 'string' ? parseISO(dateString) : dateString
    return format(date, 'MMM d, yyyy')
  } catch (error) {
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
