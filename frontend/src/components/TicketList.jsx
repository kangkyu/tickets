import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Calendar, MapPin, CheckCircle, Clock, XCircle, QrCode, Download, Mail } from 'lucide-react'
import { useUserTickets } from '../hooks/useTickets'
import { formatEventDate, formatPrice, formatSatsToUSD } from '../utils/formatters'

const TicketList = () => {
  const [selectedStatus, setSelectedStatus] = useState('all')
  
  // Mock user ID - in real app this would come from auth context
  const userId = 'user123'
  const { data: tickets = [], isLoading, error } = useUserTickets(userId)

  // Filter tickets by status
  const filteredTickets = tickets.filter(ticket => {
    if (selectedStatus === 'all') return true
    return ticket.status === selectedStatus
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

  // Loading state
  if (isLoading) {
    return (
      <div className="space-y-8">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-900 mb-4">My Tickets</h1>
          <p className="text-gray-600">View and manage your event tickets</p>
        </div>
        
        <div className="space-y-4">
          <div className="loading-skeleton h-10 w-48 rounded-lg"></div>
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

  // Error state
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

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">My Tickets</h1>
        <p className="text-gray-600">View and manage your event tickets</p>
      </div>

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
                    {tickets.filter(t => t.status === status).length}
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
            const statusInfo = getStatusInfo(ticket.status)
            const Icon = statusInfo.icon
            
            return (
              <div key={ticket.id} className="card hover:shadow-lg transition-shadow duration-200">
                {/* Event Image */}
                <div className="relative mb-4">
                  <img
                    src={ticket.eventImageUrl || '/placeholder-event.jpg'}
                    alt={ticket.eventTitle}
                    className="w-full h-48 object-cover rounded-lg"
                    onError={(e) => {
                      e.target.src = '/placeholder-event.jpg'
                    }}
                  />
                  
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
                    {ticket.eventTitle}
                  </h3>
                  
                  <div className="space-y-2 text-sm text-gray-600">
                    <div className="flex items-center gap-2">
                      <Calendar className="w-4 h-4" />
                      <span>{formatEventDate(ticket.eventDate)}</span>
                    </div>
                    
                    {ticket.eventLocation && (
                      <div className="flex items-center gap-2">
                        <MapPin className="w-4 h-4" />
                        <span>{ticket.eventLocation}</span>
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
                      <span className="text-gray-500">Price:</span>
                      <span className="font-medium text-gray-900">
                        {formatPrice(ticket.price)}
                      </span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-500">USD Value:</span>
                      <span className="text-gray-600">
                        ≈ ${formatSatsToUSD(ticket.price)}
                      </span>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="space-y-2">
                    {ticket.status === 'paid' && (
                      <>
                        <button className="w-full btn-primary">
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
                    
                    {ticket.status === 'pending' && (
                      <Link
                        to={`/tickets/${ticket.id}/payment`}
                        className="w-full btn-uma"
                      >
                        Complete Payment
                      </Link>
                    )}
                    
                    {ticket.status === 'expired' && (
                      <Link
                        to={`/events/${ticket.eventId}`}
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
          <h3 className="text-xl font-semibold text-gray-900 mb-2">No tickets found</h3>
          <p className="text-gray-600 mb-4">
            {selectedStatus === 'all' 
              ? "You haven't purchased any tickets yet."
              : `No ${selectedStatus} tickets found.`
            }
          </p>
          <Link to="/" className="btn-primary">
            Browse Events
          </Link>
        </div>
      )}

      {/* Empty State for No Tickets */}
      {tickets.length === 0 && (
        <div className="text-center py-12">
          <div className="text-gray-400 text-6xl mb-4">🎫</div>
          <h3 className="text-xl font-semibold text-gray-900 mb-2">No tickets yet</h3>
          <p className="text-gray-600 mb-4">
            Start exploring events and purchase your first ticket with Lightning payments!
          </p>
          <Link to="/" className="btn-primary">
            Discover Events
          </Link>
        </div>
      )}
    </div>
  )
}

export default TicketList
