import { z } from 'zod'

export const purchaseTicketSchema = z.object({
  eventId: z.number().min(1, 'Event ID is required'),
  userName: z.string().min(1, 'Name is required').max(100, 'Name is too long'),
  userEmail: z.string().email('Invalid email address'),
  umaAddress: z.string()
    .min(1, 'UMA address is required')
    .regex(/^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/, 'Invalid UMA format'),
  quantity: z.number().min(1, 'Quantity must be at least 1').max(10, 'Maximum 10 tickets per purchase'),
})

export const eventSearchSchema = z.object({
  query: z.string().optional(),
  startDate: z.date().optional(),
  endDate: z.date().optional(),
  maxPrice: z.number().positive('Price must be positive').optional(),
  category: z.string().optional(),
})

export const userProfileSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name is too long'),
  email: z.string().email('Invalid email address'),
  umaAddress: z.string()
    .min(1, 'UMA address is required')
    .regex(/^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/, 'Invalid UMA format'),
})

export const ticketValidationSchema = z.object({
  ticketId: z.string().min(1, 'Ticket ID is required'),
  eventId: z.number().min(1, 'Event ID is required'),
})

// Helper function to validate UMA address format
export const isValidUmaAddress = (address) => {
  if (!address) return false
  const umaRegex = /^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
  return umaRegex.test(address)
}

// Helper function to validate email format
export const isValidEmail = (email) => {
  if (!email) return false
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}
