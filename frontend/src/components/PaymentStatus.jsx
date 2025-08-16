import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { CheckCircle, Clock, XCircle, AlertCircle, RefreshCw } from 'lucide-react'
import { usePaymentPolling } from '../hooks/usePayments'
import { useTicket } from '../hooks/useTickets'
import QRCodeDisplay from './QRCodeDisplay'
import { formatPrice, formatSatsToUSD } from '../utils/formatters'

const PaymentStatus = () => {
  const { ticketId } = useParams()
  const [timeRemaining, setTimeRemaining] = useState(300) // 5 minutes in seconds
  
  // Get ticket and payment data
  const { data: ticket, isLoading: ticketLoading } = useTicket(ticketId)
  const { 
    paymentStatus, 
    error: paymentError, 
    isLoading: paymentLoading,
    hasTimedOut,
    isPending,
    isPaid,
    isExpired
  } = usePaymentPolling(ticket?.invoiceId, (status) => {
    // Callback when payment status changes
  })

  // Countdown timer for payment
  useEffect(() => {
    if (!isPending || isPaid || isExpired) return

    const interval = setInterval(() => {
      setTimeRemaining(prev => {
        if (prev <= 1) {
          clearInterval(interval)
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(interval)
  }, [isPending, isPaid, isExpired])

  // Format time remaining
  const formatTimeRemaining = (seconds) => {
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
  }

  // Loading state
  if (ticketLoading || paymentLoading) {
    return (
      <div className="text-center py-12">
        <div className="loading-skeleton w-16 h-16 rounded-full mx-auto mb-4"></div>
        <div className="loading-skeleton h-6 w-48 mx-auto mb-2"></div>
        <div className="loading-skeleton h-4 w-64 mx-auto"></div>
      </div>
    )
  }

  // Error state
  if (paymentError || !ticket) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">⚠️</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Payment Error</h2>
        <p className="text-gray-600 mb-4">
          {paymentError?.message || 'Unable to load payment information'}
        </p>
        <Link to="/" className="btn-primary">
          Back to Events
        </Link>
      </div>
    )
  }

  // Payment completed successfully
  if (isPaid) {
    return (
      <div className="text-center py-12">
        <div className="text-green-500 text-6xl mb-4">✅</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Payment Successful!</h2>
        <p className="text-gray-600 mb-6">
          Your ticket has been confirmed and is ready for use.
        </p>
        
        <div className="bg-green-50 p-6 rounded-lg border border-green-200 max-w-md mx-auto mb-6">
          <h3 className="font-semibold text-green-900 mb-2">Ticket Details</h3>
          <div className="text-sm text-green-800 space-y-1">
            <p><strong>Event:</strong> {ticket.eventTitle}</p>
            <p><strong>Date:</strong> {ticket.eventDate}</p>
            <p><strong>Ticket ID:</strong> {ticket.id}</p>
          </div>
        </div>
        
        <div className="space-y-3">
          <Link to={`/tickets`} className="btn-primary">
            View My Tickets
          </Link>
          <Link to="/" className="btn-secondary">
            Browse More Events
          </Link>
        </div>
      </div>
    )
  }

  // Payment expired or timed out
  if (isExpired || hasTimedOut) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">⏰</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Payment Expired</h2>
        <p className="text-gray-600 mb-6">
          The payment window has expired. Please try purchasing again.
        </p>
        
        <div className="space-y-3">
          <Link to={`/events/${ticket.eventId}`} className="btn-primary">
            Try Again
          </Link>
          <Link to="/" className="btn-secondary">
            Back to Events
          </Link>
        </div>
      </div>
    )
  }

  // Payment pending - show QR code and status
  return (
    <div className="max-w-2xl mx-auto space-y-8">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Complete Your Payment</h1>
        <p className="text-gray-600">
          Scan the QR code below with your Lightning wallet to complete your ticket purchase
        </p>
      </div>

      {/* Payment Status Card */}
      <div className="card">
        <div className="text-center space-y-4">
          {/* Status Icon */}
          <div className="flex justify-center">
            {isPending && (
              <div className="w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center">
                <Clock className="w-8 h-8 text-blue-600" />
              </div>
            )}
          </div>

          {/* Status Text */}
          <div>
            <h3 className="text-lg font-semibold text-gray-900 mb-1">
              {isPending ? 'Payment Pending' : 'Processing Payment'}
            </h3>
            <p className="text-gray-600">
              {isPending 
                ? 'Waiting for payment confirmation on the Lightning Network'
                : 'Verifying your payment...'
              }
            </p>
          </div>

          {/* Timer */}
          {isPending && (
            <div className="bg-orange-50 p-4 rounded-lg border border-orange-200">
              <div className="flex items-center justify-center gap-2 mb-2">
                <Clock className="w-5 h-5 text-orange-600" />
                <span className="text-orange-800 font-medium">Time Remaining</span>
              </div>
              <div className="text-2xl font-bold text-orange-900">
                {formatTimeRemaining(timeRemaining)}
              </div>
              <p className="text-sm text-orange-700 mt-1">
                Complete payment before time expires
              </p>
            </div>
          )}

          {/* Payment Amount */}
          <div className="bg-gray-50 p-4 rounded-lg">
            <p className="text-sm text-gray-500 mb-1">Payment Amount</p>
            <div className="text-2xl font-bold text-gray-900">
              {formatPrice(ticket.price)}
            </div>
            <div className="text-sm text-gray-500">
              ≈ ${formatSatsToUSD(ticket.price)}
            </div>
          </div>
        </div>
      </div>

      {/* QR Code Display */}
      {ticket.invoice && (
        <div className="card">
          <QRCodeDisplay
            bolt11Invoice={ticket.invoice}
            amount={ticket.price}
            onCopy={(invoice) => copyToClipboard(invoice)}
          />
        </div>
      )}

      {/* Payment Instructions */}
      <div className="card bg-blue-50 border-blue-200">
        <h3 className="font-semibold text-blue-900 mb-3">Payment Instructions</h3>
        <div className="space-y-2 text-sm text-blue-800">
          <p>• Use any Lightning Network compatible wallet</p>
          <p>• Popular options: Phoenix, Breez, BlueWallet, or Strike</p>
          <p>• Payment is typically confirmed within seconds</p>
          <p>• Your ticket will be available immediately after payment</p>
        </div>
      </div>

      {/* Troubleshooting */}
      <div className="card bg-yellow-50 border-yellow-200">
        <h3 className="font-semibold text-yellow-900 mb-3">Having trouble?</h3>
        <div className="space-y-2 text-sm text-yellow-800">
          <p>• Make sure your wallet has sufficient balance</p>
          <p>• Check that your wallet supports Lightning payments</p>
          <p>• If payment fails, you can try again</p>
          <p>• Contact support if you need assistance</p>
        </div>
      </div>

      {/* Cancel Option */}
      <div className="text-center">
        <Link to={`/events/${ticket.eventId}`} className="text-gray-500 hover:text-gray-700">
          Cancel and return to event
        </Link>
      </div>
    </div>
  )
}

export default PaymentStatus
