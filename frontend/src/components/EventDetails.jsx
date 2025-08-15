import { useParams, Link, Navigate } from 'react-router-dom'
import { ArrowLeft, Calendar, MapPin, Users, Clock, Share2, Heart } from 'lucide-react'
import { useEvent } from '../hooks/useEvents'
import { formatEventDate, formatPrice, formatSatsToUSD } from '../utils/formatters'

const EventDetails = () => {
  const { eventId } = useParams()
  const { data: event, isLoading, error } = useEvent(eventId)

  // Loading state
  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="loading-skeleton h-8 w-32 rounded"></div>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-6">
            <div className="loading-skeleton h-96 w-full rounded-lg"></div>
            <div className="space-y-4">
              <div className="loading-skeleton h-8 w-3/4 rounded"></div>
              <div className="loading-skeleton h-4 w-full rounded"></div>
              <div className="loading-skeleton h-4 w-2/3 rounded"></div>
            </div>
          </div>
          <div className="space-y-4">
            <div className="loading-skeleton h-32 w-full rounded-lg"></div>
            <div className="loading-skeleton h-12 w-full rounded-lg"></div>
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
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Event not found</h2>
        <p className="text-gray-600 mb-4">{error.message}</p>
        <Link to="/" className="btn-primary">
          Back to Events
        </Link>
      </div>
    )
  }

  // Event not found
  if (!event) {
    return <Navigate to="/" replace />
  }

  const handleShare = async () => {
    if (navigator.share) {
      try {
        await navigator.share({
          title: event.title,
          text: event.description,
          url: window.location.href,
        })
      } catch (error) {
        console.log('Error sharing:', error)
      }
    } else {
      // Fallback: copy to clipboard
      navigator.clipboard.writeText(window.location.href)
      // You could add a toast notification here
    }
  }

  const isEventPassed = new Date(event.date) < new Date()
  const isSoldOut = event.availableTickets === 0

  return (
    <div className="space-y-6">
      {/* Back Button */}
      <Link
        to="/"
        className="inline-flex items-center text-gray-600 hover:text-gray-900 transition-colors"
      >
        <ArrowLeft className="w-4 h-4 mr-2" />
        Back to Events
      </Link>

      {/* Event Header */}
      <div className="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 mb-2">{event.title}</h1>
          <div className="flex items-center gap-4 text-gray-600">
            <span className="flex items-center">
              <Calendar className="w-4 h-4 mr-2" />
              {formatEventDate(event.date)}
            </span>
            {event.location && (
              <span className="flex items-center">
                <MapPin className="w-4 h-4 mr-2" />
                {event.location}
              </span>
            )}
          </div>
        </div>
        
        <div className="flex items-center gap-3">
          <button
            onClick={handleShare}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            title="Share event"
          >
            <Share2 className="w-5 h-5" />
          </button>
          <button
            className="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
            title="Add to favorites"
          >
            <Heart className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Event Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Event Image */}
          <div className="relative">
            <div className="w-full h-96 bg-gradient-to-br from-uma-500 to-uma-700 rounded-lg flex items-center justify-center">
              <div className="text-center text-white">
                <Calendar className="w-24 h-24 mx-auto mb-4 opacity-80" />
                <h3 className="text-2xl font-bold mb-2">{event.title}</h3>
                <p className="text-lg opacity-80">Virtual Event</p>
              </div>
            </div>
          </div>

          {/* Event Description */}
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-gray-900">About this event</h2>
            <p className="text-gray-700 leading-relaxed">{event.description}</p>
            
            {event.additionalInfo && (
              <div className="bg-gray-50 p-4 rounded-lg">
                <h3 className="font-medium text-gray-900 mb-2">Additional Information</h3>
                <p className="text-gray-700">{event.additionalInfo}</p>
              </div>
            )}
          </div>

          {/* Event Details */}
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-gray-900">Event Details</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="flex items-center p-3 bg-gray-50 rounded-lg">
                <Calendar className="w-5 h-5 text-uma-600 mr-3" />
                <div>
                  <p className="text-sm text-gray-500">Date & Time</p>
                  <p className="font-medium text-gray-900">{formatEventDate(event.date)}</p>
                </div>
              </div>
              
              {event.location && (
                <div className="flex items-center p-3 bg-gray-50 rounded-lg">
                  <MapPin className="w-5 h-5 text-uma-600 mr-3" />
                  <div>
                    <p className="text-sm text-gray-500">Location</p>
                    <p className="font-medium text-gray-900">{event.location}</p>
                  </div>
                </div>
              )}
              
              <div className="flex items-center p-3 bg-gray-50 rounded-lg">
                <Users className="w-5 h-5 text-uma-600 mr-3" />
                <div>
                  <p className="text-sm text-gray-500">Available Tickets</p>
                  <p className="font-medium text-gray-900">{event.availableTickets}</p>
                </div>
              </div>
              
              {event.organizer && (
                <div className="flex items-center p-3 bg-gray-50 rounded-lg">
                  <div className="w-5 h-5 bg-uma-600 rounded-full mr-3 flex items-center justify-center">
                    <span className="text-white text-xs font-bold">
                      {event.organizer.charAt(0).toUpperCase()}
                    </span>
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Organizer</p>
                    <p className="font-medium text-gray-900">{event.organizer}</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Sidebar - Purchase Card */}
        <div className="lg:col-span-1">
          <div className="card sticky top-8">
            <div className="space-y-4">
              {/* Price */}
              <div className="text-center">
                <div className="text-3xl font-bold text-uma-600">{formatPrice(event.price)}</div>
                <div className="text-sm text-gray-500">
                  ≈ ${formatSatsToUSD(event.price)}
                </div>
              </div>

              {/* Status */}
              <div className="text-center">
                {isEventPassed ? (
                  <span className="inline-block bg-red-100 text-red-800 px-3 py-1 rounded-full text-sm font-medium">
                    Event has passed
                  </span>
                ) : isSoldOut ? (
                  <span className="inline-block bg-red-100 text-red-800 px-3 py-1 rounded-full text-sm font-medium">
                    Sold Out
                  </span>
                ) : event.availableTickets <= 5 ? (
                  <span className="inline-block bg-orange-100 text-orange-800 px-3 py-1 rounded-full text-sm font-medium">
                    Only {event.availableTickets} tickets left!
                  </span>
                ) : (
                  <span className="inline-block bg-green-100 text-green-800 px-3 py-1 rounded-full text-sm font-medium">
                    Tickets Available
                  </span>
                )}
              </div>

              {/* Purchase Button */}
              {!isEventPassed && !isSoldOut ? (
                <Link
                  to={`/events/${event.id}/purchase`}
                  className="btn-uma w-full text-center"
                >
                  Purchase Tickets
                </Link>
              ) : (
                <button
                  disabled
                  className="w-full bg-gray-300 text-gray-500 font-medium py-3 px-4 rounded-lg cursor-not-allowed"
                >
                  {isEventPassed ? 'Event has passed' : 'Sold Out'}
                </button>
              )}

              {/* Additional Info */}
              <div className="text-xs text-gray-500 text-center space-y-2">
                <p>• Secure Lightning Network payment</p>
                <p>• Instant ticket delivery</p>
                <p>• No additional fees</p>
                {event.refundPolicy && (
                  <p>• {event.refundPolicy}</p>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default EventDetails
