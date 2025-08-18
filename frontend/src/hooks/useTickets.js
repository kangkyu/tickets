import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ticketsAPI } from '../services/api'

export const useTicket = (ticketId) => {
  return useQuery({
    queryKey: ['ticket', ticketId],
    queryFn: () => ticketsAPI.getById(ticketId),
    enabled: !!ticketId,
    staleTime: 2 * 60 * 1000, // 2 minutes
  })
}

export const useTicketStatus = (ticketId) => {
  return useQuery({
    queryKey: ['ticketStatus', ticketId],
    queryFn: () => ticketsAPI.getStatus(ticketId),
    enabled: !!ticketId,
    refetchInterval: 10000, // Poll every 10 seconds
    staleTime: 0, // Always consider stale for real-time updates
  })
}

export const useUserTickets = (userId) => {
  return useQuery({
    queryKey: ['userTickets', userId],
    queryFn: () => ticketsAPI.getUserTickets(userId),
    enabled: !!userId && userId > 0,
    staleTime: 2 * 60 * 1000,
    retry: false,
  })
}

export const useTicketPurchase = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ticketsAPI.purchase,
    onSuccess: (data) => {
      // Invalidate user tickets
      if (data.userId) {
        queryClient.invalidateQueries({ queryKey: ['userTickets', data.userId] })
      }
      
      // Invalidate event data if it affects availability
      if (data.eventId) {
        queryClient.invalidateQueries({ queryKey: ['event', data.eventId] })
      }
    },
    onError: (error) => {
      console.error('Failed to purchase ticket:', error)
    },
  })
}

export const useTicketValidation = () => {
  return useMutation({
    mutationFn: ticketsAPI.validate,
    onError: (error) => {
      console.error('Failed to validate ticket:', error)
    },
  })
}
