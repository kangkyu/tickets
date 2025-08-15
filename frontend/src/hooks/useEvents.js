import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { eventsAPI } from '../services/api'

export const useEvents = (filters = {}) => {
  return useQuery({
    queryKey: ['events', filters],
    queryFn: () => eventsAPI.getAll(filters),
    staleTime: 5 * 60 * 1000, // 5 minutes
    refetchOnWindowFocus: false,
  })
}

export const useEvent = (eventId) => {
  return useQuery({
    queryKey: ['event', eventId],
    queryFn: () => eventsAPI.getById(eventId),
    enabled: !!eventId,
    staleTime: 5 * 60 * 1000,
  })
}

export const useEventSearch = (query, filters = {}) => {
  return useQuery({
    queryKey: ['eventSearch', query, filters],
    queryFn: () => eventsAPI.search(query, filters),
    enabled: !!query || Object.keys(filters).length > 0,
    staleTime: 2 * 60 * 1000, // 2 minutes for search results
  })
}

export const useEventMutation = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: eventsAPI.create,
    onSuccess: () => {
      // Invalidate and refetch events list
      queryClient.invalidateQueries({ queryKey: ['events'] })
    },
    onError: (error) => {
      console.error('Failed to create event:', error)
    },
  })
}
