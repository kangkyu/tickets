import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Calendar, MapPin, CheckCircle, Clock, XCircle, QrCode, Download, Mail } from 'lucide-react'
import { useUserTickets } from '../hooks/useTickets'
import { formatEventDate, formatPrice, formatSatsToUSD } from '../utils/formatters'
import QRCodeDisplay from './QRCodeDisplay'
import { useAuth } from '../contexts/AuthContext'
import config from '../config/api'

const TicketList = () => {
  const { user, isAuthenticated } = useAuth()
  const location = useLocation()
  const { data: response, isLoading, error, refetch } = useUserTickets(isAuthenticated && user?.id ? user.id : null)
  
  // Extract tickets from the response structure
  const tickets = response?.data || response || []
  
  const [selectedStatus, setSelectedStatus] = useState('all')
  const [selectedTicket, setSelectedTicket] = useState(null)
  const [refreshingTickets, setRefreshingTickets] = useState(new Set())

  // Get success message from navigation state
  const { successMessage } = location.state || {}

  // Show loading or redirect if not authenticated
  if (!isAuthenticated) {
    return (
      <div className="text-center py-12">
        <div className="text-gray-400 text-6xl mb-4">🔒</div>
        <h3 className="text-xl font-semibold text-gray-900 mb-2">Authentication Required</h3>
        <p className="text-gray-600 mb-4">
          Please sign in to view your tickets.
        </p>
        <Link to="/login" className="btn-primary">
          Sign In
        </Link>
      </div>
    )
  }

  // Show loading if user ID is not available yet
  if (!user?.id) {
    return (
      <div className="text-center py-12">
        <div className="loading-skeleton h-8 w-48 mx-auto mb-4"></div>
        <div className="loading-skeleton h-4 w-64 mx-auto"></div>
      </div>
    )
  }

  // Show loading state while fetching tickets
  if (isLoading) {
    return (
      <div className="space-y-8">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-900 mb-4">My Tickets</h1>
          <p className="text-gray-600">View and manage your event tickets</p>
        </div>
        
        <div className="space-y-4">
          <div className="loading-skeleton h-10 w-48 rounded-lg mx-auto"></div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="card">
                <div className="loading-skeleton h-32 w-full rounded-lg mb-4"></div>
                <div className="space-y-3">
                  <div className="loading-skeleton h-6 w-3/4 rounded"></div>
                  <div className="loading-skeleton h-4 w-full rounded"></div>
                  <div className="loading-skeleton h-4 w-2/3 rounded"></div>
                  <div className="loading-skeleton h-10 w-full rounded-lg"></div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  // Show error state if API call failed
  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">⚠️</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Unable to load tickets</h2>
        <p className="text-gray-600 mb-4">{error.message}</p>
        <button
          onClick={() => window.location.reload()}
          className="btn-primary"
        >
          Try Again
        </button>
      </div>
    )
  }

  // Filter tickets by status
  const filteredTickets = (Array.isArray(tickets) ? tickets : []).filter(ticket => {
    if (selectedStatus === 'all') return true
    return ticket.payment_status === selectedStatus
  })

  // Get status icon and color
  const getStatusInfo = (status) => {
    switch (status) {
      case 'paid':
        return {
          icon: CheckCircle,
          color: 'text-green-600',
          bgColor: 'bg-green-100',
          borderColor: 'border-green-200',
          label: 'Confirmed'
        }
      case 'pending':
        return {
          icon: Clock,
          color: 'text-orange-600',
          bgColor: 'bg-orange-100',
          borderColor: 'border-orange-200',
          label: 'Payment Pending'
        }
      case 'expired':
        return {
          icon: XCircle,
          color: 'text-red-600',
          bgColor: 'bg-red-100',
          borderColor: 'border-red-200',
          label: 'Expired'
        }
      default:
        return {
          icon: Clock,
          color: 'text-gray-600',
          bgColor: 'bg-gray-100',
          borderColor: 'border-gray-200',
          label: 'Unknown'
        }
    }
  }

  const handleShowQRCode = (ticket) => {
    setSelectedTicket(ticket)
  }

  const handleCloseQRCode = () => {
    setSelectedTicket(null)
  }

  const handleRefreshTicketStatus = async (ticketId) => {
    try {
      // Add ticket to refreshing set
      setRefreshingTickets(prev => new Set(prev).add(ticketId))
      
      // Use the existing ticket status endpoint
      const response = await fetch(`${config.apiUrl}/api/tickets/${ticketId}/status`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
          'Content-Type': 'application/json'
        }
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      // Refetch all tickets to get updated status
      await refetch()
      
    } catch (error) {
      console.error('Error refreshing ticket status:', error)
      // Don't show alert, just log the error
    } finally {
      // Remove ticket from refreshing set
      setRefreshingTickets(prev => {
        const newSet = new Set(prev)
        newSet.delete(ticketId)
        return newSet
      })
    }
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">My Tickets</h1>
        <p className="text-gray-600">View and manage your event tickets</p>
      </div>

      {/* Success Message */}
      {successMessage && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 max-w-2xl mx-auto">
          <div className="flex items-center gap-3">
            <CheckCircle className="w-6 h-6 text-green-600" />
            <div className="text-left">
              <p className="text-green-800 font-medium">{successMessage}</p>
              <p className="text-green-700 text-sm mt-1">
                Your ticket information is displayed below
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Status Filter */}
      <div className="flex justify-center">
        <div className="flex bg-gray-100 p-1 rounded-lg">
          {['all', 'paid', 'pending', 'expired'].map((status) => {
            const isActive = selectedStatus === status
            const statusInfo = getStatusInfo(status)
            const Icon = statusInfo.icon
            
            return (
              <button
                key={status}
                onClick={() => setSelectedStatus(status)}
                className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-white text-gray-900 shadow-sm'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                <Icon className="w-4 h-4" />
                <span className="capitalize">
                  {status === 'all' ? 'All Tickets' : statusInfo.label}
                </span>
                {status !== 'all' && (
                  <span className="bg-gray-200 text-gray-700 px-2 py-0.5 rounded-full text-xs">
                    {Array.isArray(tickets) ? tickets.filter(t => t.payment_status === status).length : 0}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      </div>

      {/* Results Count */}
      <div className="text-center">
        <p className="text-gray-600">
          {filteredTickets.length} ticket{filteredTickets.length !== 1 ? 's' : ''} found
        </p>
      </div>

      {/* Tickets Grid */}
      {filteredTickets.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredTickets.map(ticket => {
            const statusInfo = getStatusInfo(ticket.payment_status)
            const Icon = statusInfo.icon
            
            return (
              <div key={ticket.id} className="card hover:shadow-lg transition-shadow duration-200">
                {/* Event Image */}
                <div className="relative mb-4">
                  <div className="w-full h-48 bg-gradient-to-br from-uma-500 to-uma-700 rounded-lg flex items-center justify-center">
                    <div className="text-center text-white">
                      <Calendar className="w-12 h-12 mx-auto mb-2 opacity-80" />
                      <p className="text-sm font-semibold">{ticket.event?.title?.split(' ')[0] || 'Event'}</p>
                      <p className="text-xs opacity-80">Event</p>
                    </div>
                  </div>
                  
                  {/* Status Badge */}
                  <div className={`absolute top-3 right-3 ${statusInfo.bgColor} ${statusInfo.borderColor} border px-3 py-1 rounded-full`}>
                    <div className="flex items-center gap-1">
                      <Icon className={`w-3 h-3 ${statusInfo.color}`} />
                      <span className={`text-xs font-medium ${statusInfo.color}`}>
                        {statusInfo.label}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Ticket Details */}
                <div className="space-y-3">
                  <h3 className="text-lg font-semibold text-gray-900">
                    {ticket.event?.title || 'Unknown Event'}
                  </h3>
                  
                  <div className="space-y-2 text-sm text-gray-600">
                    <div className="flex items-center gap-2">
                      <Calendar className="w-4 h-4" />
                      <span>{ticket.event?.start_time ? formatEventDate(new Date(ticket.event.start_time)) : 'Date TBD'}</span>
                    </div>
                    
                    {ticket.event?.location && (
                      <div className="flex items-center gap-2">
                        <MapPin className="w-4 h-4" />
                        <span>{ticket.event.location}</span>
                      </div>
                    )}
                  </div>

                  {/* Ticket Info */}
                  <div className="bg-gray-50 p-3 rounded-lg space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-500">Ticket ID:</span>
                      <span className="font-mono text-gray-900">{ticket.id}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-500">Ticket Code:</span>
                      <span className="font-mono text-gray-900">{ticket.ticket_code}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-500">Price:</span>
                      <span className="font-medium text-gray-900">
                        {ticket.event?.price_sats ? formatPrice(ticket.event.price_sats) : 'N/A'}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-500">USD Value:</span>
                      <span className="text-gray-600">
                        {ticket.event?.price_sats ? `≈ $${formatSatsToUSD(ticket.event.price_sats)}` : 'N/A'}
                      </span>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="space-y-2">
                    {ticket.payment_status === 'paid' && (
                      <>
                        <button 
                          onClick={() => handleShowQRCode(ticket)}
                          className="w-full btn-primary"
                        >
                          <QrCode className="w-4 h-4 mr-2" />
                          Show QR Code
                        </button>
                        <div className="flex gap-2">
                          <button className="flex-1 btn-secondary text-sm">
                            <Download className="w-4 h-4 mr-2" />
                            Download
                          </button>
                          <button className="flex-1 btn-secondary text-sm">
                            <Mail className="w-4 h-4 mr-2" />
                            Email
                          </button>
                        </div>
                      </>
                    )}
                    
                    {ticket.payment_status === 'pending' && (
                      <div className="w-full p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                        <div className="text-center">
                          <Clock className="w-5 h-5 text-yellow-600 mx-auto mb-2" />
                          <p className="text-sm text-yellow-800 font-medium">Payment Pending</p>
                          <p className="text-xs text-yellow-700 mt-1">
                            Complete your Lightning Network payment to confirm your ticket
                          </p>
                          {ticket.payment && (
                            <div className="mt-2 text-xs text-yellow-700 space-y-1">
                              <div className="flex justify-between">
                                <span>Amount:</span>
                                <span className="font-medium">{formatPrice(ticket.payment.amount_sats)}</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Invoice ID:</span>
                                <span className="font-mono">{ticket.payment.invoice_id}</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Status:</span>
                                <span className="font-medium">{ticket.payment.status}</span>
                              </div>
                            </div>
                          )}
                          <div className="mt-3">
                            <button
                              onClick={() => handleRefreshTicketStatus(ticket.id)}
                              disabled={refreshingTickets.has(ticket.id)}
                              className="btn-primary text-xs px-3 py-1 disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                              {refreshingTickets.has(ticket.id) ? '🔄 Refreshing...' : '🔄 Refresh Status'}
                            </button>
                          </div>
                        </div>
                      </div>
                    )}
                    
                    {ticket.payment_status === 'expired' && (
                      <Link
                        to={`/events/${ticket.event?.id}`}
                        className="w-full btn-secondary"
                      >
                        View Event
                      </Link>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      ) : (
        <div className="text-center py-12">
          <div className="text-gray-400 text-6xl mb-4">🎫</div>
          <h3 className="text-xl font-semibold text-gray-900 mb-2">
            {selectedStatus === 'all' ? 'No tickets yet' : `No ${selectedStatus} tickets found`}
          </h3>
          <p className="text-gray-600 mb-4">
            {selectedStatus === 'all' 
              ? "Start exploring events and purchase your first ticket with Lightning payments!"
              : `No tickets with ${selectedStatus} status found. Try changing the filter or browse events.`
            }
          </p>
          <Link to="/" className="btn-primary">
            {selectedStatus === 'all' ? 'Discover Events' : 'Browse Events'}
          </Link>
        </div>
      )}

      {/* QR Code Modal */}
      {selectedTicket && (
        <QRCodeDisplay
          ticket={selectedTicket}
          onClose={handleCloseQRCode}
        />
      )}
    </div>
  )
}

export default TicketList
