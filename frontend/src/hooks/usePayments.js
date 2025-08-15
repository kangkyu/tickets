import { useQuery, useMutation } from '@tanstack/react-query'
import { paymentsAPI, apiUtils } from '../services/api'

export const usePaymentStatus = (invoiceId) => {
  return useQuery({
    queryKey: ['paymentStatus', invoiceId],
    queryFn: () => paymentsAPI.getStatus(invoiceId),
    enabled: !!invoiceId,
    refetchInterval: 10000, // Poll every 10 seconds
    staleTime: 0, // Always consider stale for real-time updates
    retry: (failureCount, error) => {
      // Don't retry on 404 (invoice not found)
      if (error.status === 404) return false
      return failureCount < 3
    },
  })
}

export const usePaymentCreation = () => {
  return useMutation({
    mutationFn: paymentsAPI.createInvoice,
    onError: (error) => {
      console.error('Failed to create payment invoice:', error)
    },
  })
}

export const usePaymentCancellation = () => {
  return useMutation({
    mutationFn: paymentsAPI.cancel,
    onError: (error) => {
      console.error('Failed to cancel payment:', error)
    },
  })
}

import React from 'react'

// Custom hook for payment polling with timeout
export const usePaymentPolling = (invoiceId, onStatusChange) => {
  const { data: paymentStatus, error, isLoading } = usePaymentStatus(invoiceId)
  
  // Effect to handle status changes
  React.useEffect(() => {
    if (paymentStatus && onStatusChange) {
      onStatusChange(paymentStatus)
    }
  }, [paymentStatus, onStatusChange])
  
  // Check if payment has timed out
  const hasTimedOut = React.useMemo(() => {
    if (!paymentStatus?.createdAt) return false
    
    const createdAt = new Date(paymentStatus.createdAt)
    const now = new Date()
    const timeDiff = now - createdAt
    
    return timeDiff > 300000 // 5 minutes
  }, [paymentStatus?.createdAt])
  
  return {
    paymentStatus,
    error,
    isLoading,
    hasTimedOut,
    isPending: paymentStatus?.status === 'pending',
    isPaid: paymentStatus?.status === 'paid',
    isExpired: paymentStatus?.status === 'expired' || hasTimedOut,
  }
}

// Hook for manual payment polling (useful for one-time checks)
export const useManualPaymentCheck = () => {
  const [isChecking, setIsChecking] = React.useState(false)
  
  const checkPayment = React.useCallback(async (invoiceId) => {
    setIsChecking(true)
    try {
      const result = await apiUtils.poll(
        () => paymentsAPI.getStatus(invoiceId),
        10000, // 10 second interval
        300000  // 5 minute timeout
      )
      return result
    } catch (error) {
      throw error
    } finally {
      setIsChecking(false)
    }
  }, [])
  
  return {
    checkPayment,
    isChecking,
  }
}
