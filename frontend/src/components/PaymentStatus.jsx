import { useState, useEffect } from 'react'
import { useParams, useLocation, useNavigate } from 'react-router-dom'
import { ArrowLeft, CheckCircle, XCircle, Clock, Copy, Download, Mail, RefreshCw } from 'lucide-react'
import { formatPrice, formatSatsToUSD } from '../utils/formatters'
import config from '../config/api'

const PaymentStatus = () => {
  const { ticketId } = useParams()
  const location = useLocation()
  const navigate = useNavigate()
  
  const [ticket, setTicket] = useState(null)
  const [payment, setPayment] = useState(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState(null)
  const [copySuccess, setCopySuccess] = useState(false)
  const [isPolling, setIsPolling] = useState(false)
  
  // Get data from navigation state
  const { invoiceId, bolt11, amountSats, umaAddress, ticketData, isFreeEvent } = location.state || {}

  // Debug logging
  console.log('PaymentStatus component data:', {
    ticketId,
    invoiceId,
    umaAddress,
    ticketData,
    locationState: location.state
  })

  useEffect(() => {
    if (ticketId) {
      fetchTicketAndPaymentStatus()
    }
  }, [ticketId])

  // Set up polling for payment status updates
  useEffect(() => {
    if (ticket?.payment_status === 'pending' && invoiceId) {
      const pollInterval = setInterval(() => {
        pollPaymentStatus()
      }, 5000) // Poll every 5 seconds

      return () => clearInterval(pollInterval)
    }
  }, [ticket?.payment_status, invoiceId])

  const fetchTicketAndPaymentStatus = async () => {
    try {
      setIsLoading(true)
      
      // Fetch ticket status
      const ticketResponse = await fetch(`${config.apiUrl}/api/tickets/${ticketId}/status`)
      if (!ticketResponse.ok) {
        throw new Error('Failed to fetch ticket status')
      }
      
      const ticketData = await ticketResponse.json()
      setTicket(ticketData.data)
      
      // Fetch payment status if we have an invoice ID
      if (invoiceId) {
        const authToken = localStorage.getItem('authToken')
        const paymentResponse = await fetch(`${config.apiUrl}/api/payments/${invoiceId}/status`, {
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'Content-Type': 'application/json'
          }
        })
        if (paymentResponse.ok) {
          const paymentData = await paymentResponse.json()
          setPayment(paymentData.data)
        } else if (paymentResponse.status === 401) {
          console.warn('Payment status requires authentication')
          // Payment status is protected, but we can still show ticket info
        } else {
          console.error('Failed to fetch payment status:', paymentResponse.status)
        }
      }
      
    } catch (err) {
      console.error('Failed to fetch ticket/payment data:', err)
      setError(err.message)
    } finally {
      setIsLoading(false)
    }
  }

  const pollPaymentStatus = async () => {
    if (isPolling) return // Prevent multiple simultaneous requests
    
    try {
      setIsPolling(true)
      
      // Fetch updated ticket status
      const ticketResponse = await fetch(`${config.apiUrl}/api/tickets/${ticketId}/status`)
      if (ticketResponse.ok) {
        const ticketData = await ticketResponse.json()
        setTicket(ticketData.data)
        
        // If payment is now complete, stop polling
        if (ticketData.data.payment_status === 'paid') {
          setIsPolling(false)
          // Show success message
          setTimeout(() => {
            navigate('/tickets', { replace: true })
          }, 2000)
        }
      }
      
      // Fetch updated payment status (with auth)
      if (invoiceId) {
        const authToken = localStorage.getItem('authToken')
        const paymentResponse = await fetch(`${config.apiUrl}/api/payments/${invoiceId}/status`, {
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'Content-Type': 'application/json'
          }
        })
        if (paymentResponse.ok) {
          const paymentData = await paymentResponse.json()
          setPayment(paymentData.data)
        }
      }
      
    } catch (err) {
      console.error('Failed to poll payment status:', err)
    } finally {
      setIsPolling(false)
    }
  }

  const copyToClipboard = async (text) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopySuccess(true)
      setTimeout(() => setCopySuccess(false), 2000)
    } catch (err) {
      console.error('Failed to copy to clipboard:', err)
    }
  }

  const downloadInvoice = () => {
    const invoiceString = bolt11 || payment?.invoice?.bolt11
    if (invoiceString) {
      const blob = new Blob([invoiceString], { type: 'text/plain' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `invoice-${invoiceId}.txt`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    }
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="text-center py-12">
        <div className="loading-skeleton w-16 h-16 rounded-full mx-auto mb-4 animate-spin"></div>
        <div className="loading-skeleton h-6 w-48 mx-auto mb-2"></div>
        <div className="loading-skeleton h-4 w-64 mx-auto"></div>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">⚠️</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Error Loading Payment</h2>
        <p className="text-gray-600 mb-4">{error}</p>
        <button onClick={() => navigate(-1)} className="btn-primary">
          Go Back
        </button>
      </div>
    )
  }

  if (!ticket) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">⚠️</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Ticket Not Found</h2>
        <p className="text-gray-600 mb-4">Unable to load ticket information</p>
        <button onClick={() => navigate(-1)} className="btn-primary">
          Go Back
        </button>
      </div>
    )
  }

  const isPaid = ticket.payment_status === 'paid'
  const isPending = ticket.payment_status === 'pending'
  const isExpired = ticket.payment_status === 'expired'
  const isFailed = ticket.payment_status === 'failed'

  return (
    <div className="max-w-4xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate(-1)}
          className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Payment Status</h1>
          <p className="text-gray-600">Ticket #{ticket.ticket_code}</p>
        </div>
      </div>

      {/* Payment Status Card */}
      <div className="card">
        <div className="text-center space-y-4">
          {/* Status Icon */}
          <div className="mx-auto">
            {isPaid && (
              <div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto">
                <CheckCircle className="w-12 h-12 text-green-600" />
              </div>
            )}
            {isPending && (
              <div className="w-20 h-20 bg-yellow-100 rounded-full flex items-center justify-center mx-auto">
                <Clock className="w-12 h-12 text-yellow-600" />
              </div>
            )}
            {isExpired && (
              <div className="w-20 h-20 bg-red-100 rounded-full flex items-center justify-center mx-auto">
                <XCircle className="w-12 h-12 text-red-600" />
              </div>
            )}
            {isFailed && (
              <div className="w-20 h-20 bg-red-100 rounded-full flex items-center justify-center mx-auto">
                <XCircle className="w-12 h-12 text-red-600" />
              </div>
            )}
          </div>

          {/* Status Text */}
          <div>
            <h2 className="text-2xl font-bold text-gray-900 mb-2">
              {isPaid && 'Payment Successful!'}
              {isPending && 'Payment Pending'}
              {isExpired && 'Payment Expired'}
              {isFailed && 'Payment Failed'}
            </h2>
            <p className="text-gray-600">
              {isPaid && 'Your ticket has been confirmed and is ready to use.'}
              {isPending && 'Please complete your Lightning Network payment to confirm your ticket.'}
              {isExpired && 'The payment window has expired. Please try purchasing again.'}
              {isFailed && 'The payment was unsuccessful. Please try again.'}
            </p>
          </div>

          {/* Payment Details */}
          {payment && (
            <div className="bg-gray-50 rounded-lg p-4 max-w-md mx-auto">
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-600">Amount:</span>
                  <span className="font-medium">
                    {formatPrice(payment.payment.amount_sats)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Status:</span>
                  <span className={`font-medium ${
                    isPaid ? 'text-green-600' : 
                    isPending ? 'text-yellow-600' : 
                    'text-red-600'
                  }`}>
                    {payment.payment.status}
                  </span>
                </div>
                {payment.payment.paid_at && (
                  <div className="flex justify-between">
                    <span className="text-gray-600">Paid at:</span>
                    <span className="font-medium">
                      {new Date(payment.payment.paid_at).toLocaleString()}
                    </span>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Fallback when payment data isn't available */}
          {!payment && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 max-w-md mx-auto">
              <div className="text-center">
                <Clock className="w-8 h-8 text-yellow-600 mx-auto mb-2" />
                <p className="text-sm text-yellow-800 font-medium">Payment Information Loading</p>
                <p className="text-xs text-yellow-700 mt-1">
                  {invoiceId ? 'Fetching payment details...' : 'No invoice ID available'}
                </p>
                {!invoiceId && (
                  <p className="text-xs text-yellow-600 mt-2">
                    This usually means the ticket purchase wasn't completed properly.
                  </p>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Payment Instructions */}
      {isPending && payment?.invoice && (
        <div className="card">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Complete Your Payment</h3>
          
          <div className="space-y-4">
            {/* UMA Address Info */}
            {umaAddress && (
              <div className="bg-uma-50 p-4 rounded-lg border border-uma-200">
                <p className="text-sm text-uma-700 mb-2 font-medium">UMA Address</p>
                <p className="text-uma-900 font-mono">{umaAddress}</p>
                <p className="text-xs text-uma-600 mt-1">
                  This is the address where your payment will be sent via the Universal Money Address protocol
                </p>
              </div>
            )}

            {/* Lightning Invoice */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium text-gray-700">Lightning Invoice (BOLT11)</label>
                <div className="flex items-center gap-2">
                  <button
                    onClick={pollPaymentStatus}
                    disabled={isPolling}
                    className="text-uma-600 hover:text-uma-700 text-sm font-medium disabled:opacity-50"
                  >
                    <RefreshCw className={`w-4 h-4 ${isPolling ? 'animate-spin' : ''}`} />
                    {isPolling ? 'Checking...' : 'Check Status'}
                  </button>
                  <button
                    onClick={() => copyToClipboard(payment.invoice.bolt11)}
                    className="text-uma-600 hover:text-uma-700 text-sm font-medium"
                  >
                    {copySuccess ? 'Copied!' : 'Copy'}
                  </button>
                </div>
              </div>
              
              {/* QR Code for Mobile Wallets */}
              <div className="text-center">
                <div className="inline-block p-4 bg-white border-2 border-gray-200 rounded-lg">
                  <img
                    src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(payment.invoice.bolt11)}`}
                    alt="Lightning Invoice QR Code"
                    className="w-48 h-48"
                  />
                </div>
                <p className="text-xs text-gray-500 mt-2">
                  Scan with your Lightning wallet app
                </p>
              </div>
              
              <div className="relative">
                <textarea
                  readOnly
                  value={payment.invoice.bolt11}
                  className="w-full p-3 border border-gray-300 rounded-lg font-mono text-sm bg-gray-50 resize-none"
                  rows={3}
                />
                <button
                  onClick={downloadInvoice}
                  className="absolute top-2 right-2 p-1 text-gray-400 hover:text-gray-600"
                  title="Download invoice"
                >
                  <Download className="w-4 h-4" />
                </button>
              </div>
              
              <div className="bg-blue-50 p-3 rounded-lg border border-blue-200">
                <h4 className="font-medium text-blue-900 mb-2">How to pay with UMA:</h4>
                <ol className="text-sm text-blue-800 space-y-1">
                  <li>1. Open your Lightning wallet app</li>
                  <li>2. Scan the QR code or paste the invoice above</li>
                  <li>3. Confirm the payment amount and UMA address</li>
                  <li>4. Complete the payment</li>
                  <li>5. Status will update automatically (or click "Check Status")</li>
                </ol>
              </div>
            </div>

            {/* Payment Expiry */}
            {payment.invoice.expires_at && (
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                <div className="flex items-center gap-2">
                  <Clock className="w-4 h-4 text-yellow-600" />
                  <p className="text-sm text-yellow-800">
                    Payment expires at {new Date(payment.invoice.expires_at).toLocaleString()}
                  </p>
                </div>
              </div>
            )}

            {/* Auto-refresh Status */}
            <div className="bg-gray-50 p-3 rounded-lg">
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-gray-600" />
                <p className="text-sm text-gray-700">
                  Payment status updates automatically every 5 seconds
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Ticket Information */}
      <div className="card">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Ticket Details</h3>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <p className="text-sm text-gray-500">Ticket Code</p>
            <p className="font-mono font-medium">{ticket.ticket_code}</p>
          </div>
          
          <div>
            <p className="text-sm text-gray-500">Payment Status</p>
            <p className={`font-medium ${
              isPaid ? 'text-green-600' : 
              isPending ? 'text-yellow-600' : 
              'text-red-600'
            }`}>
              {ticket.payment_status}
            </p>
          </div>
          
          {ticketData && (
            <>
              <div>
                <p className="text-sm text-gray-500">Event</p>
                <p className="font-medium">{ticketData.eventTitle || 'N/A'}</p>
              </div>
              
              <div>
                <p className="text-sm text-gray-500">Quantity</p>
                <p className="font-medium">{ticketData.quantity || 1}</p>
              </div>
            </>
          )}
        </div>

        {/* Debug Information (only in development) */}
        {import.meta.env.DEV && (
          <div className="mt-6 p-4 bg-gray-100 rounded-lg">
            <h4 className="text-sm font-medium text-gray-700 mb-2">Debug Info</h4>
            <div className="text-xs text-gray-600 space-y-1">
              <div>Ticket ID: {ticketId}</div>
              <div>Invoice ID: {invoiceId || 'Not available'}</div>
              <div>UMA Address: {umaAddress || 'Not available'}</div>
              <div>Payment Data: {payment ? 'Available' : 'Not available'}</div>
              <div>Ticket Data: {ticketData ? 'Available' : 'Not available'}</div>
            </div>
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex flex-col sm:flex-row gap-3 justify-center">
        {isPaid ? (
          <button
            onClick={() => navigate('/tickets')}
            className="btn-uma"
          >
            View My Tickets
          </button>
        ) : isPending ? (
          <div className="text-center">
            <p className="text-sm text-gray-600 mb-3">
              Payment status will update automatically
            </p>
            <button
              onClick={() => window.location.reload()}
              className="btn-secondary"
            >
              Refresh Status
            </button>
          </div>
        ) : (
          <button
            onClick={() => navigate(-1)}
            className="btn-primary"
          >
            Try Again
          </button>
        )}
        
        <button
          onClick={() => navigate('/')}
          className="btn-secondary"
        >
          Back to Events
        </button>
      </div>
    </div>
  )
}

export default PaymentStatus
