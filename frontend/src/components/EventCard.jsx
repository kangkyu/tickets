import { Link } from 'react-router-dom'
import { Calendar, Users } from 'lucide-react'
import { formatEventDateShort, formatPrice, truncateText } from '../utils/formatters'

const EventCard = ({ event }) => {
  const {
    id,
    title,
    description,
    start_time,
    end_time,
    price_sats,
    capacity,
    is_active,
    stream_url
  } = event

  // For now, assume all capacity is available (no ticket counting implemented yet)
  const availableTickets = capacity
  const price = price_sats
  const date = start_time

  return (
    <div className="card hover:shadow-lg transition-shadow duration-200 group">
      {/* Event Image */}
      <div className="relative mb-4">
        <div className="w-full h-48 bg-gradient-to-br from-uma-500 to-uma-700 rounded-lg flex items-center justify-center group-hover:scale-105 transition-transform duration-200">
          <div className="text-center text-white">
            <Calendar className="w-16 h-16 mx-auto mb-2 opacity-80" />
            <p className="text-lg font-semibold">{title.split(' ')[0]}</p>
            <p className="text-sm opacity-80">Event</p>
          </div>
        </div>
        {availableTickets <= 5 && availableTickets > 0 && (
          <span className="absolute top-3 right-3 bg-orange-500 text-white text-xs font-medium px-2 py-1 rounded-full">
            Only {availableTickets} left!
          </span>
        )}
        {availableTickets === 0 && (
          <span className="absolute top-3 right-3 bg-red-500 text-white text-xs font-medium px-2 py-1 rounded-full">
            Sold Out
          </span>
        )}
      </div>

      {/* Event Details */}
      <div className="space-y-3">
        <h3 className="text-lg font-semibold text-gray-900 group-hover:text-uma-600 transition-colors duration-200">
          {title}
        </h3>
        
        <p className="text-gray-600 text-sm line-clamp-2">
          {truncateText(description, 120)}
        </p>

        {/* Event Meta Information */}
        <div className="space-y-2">
          <div className="flex items-center text-sm text-gray-500">
            <Calendar className="w-4 h-4 mr-2" />
            <span>{formatEventDateShort(date)}</span>
          </div>
          
          <div className="flex items-center justify-between">
            <div className="flex items-center text-sm text-gray-500">
              <Users className="w-4 h-4 mr-2" />
              <span>{availableTickets} tickets available</span>
            </div>
            
            <div className="text-lg font-bold text-uma-600">
              {formatPrice(price)}
            </div>
          </div>
        </div>

        {/* Action Button */}
        <div className="pt-2">
          {availableTickets > 0 ? (
            <Link
              to={`/events/${id}`}
              className="btn-uma w-full text-center"
            >
              View Details
            </Link>
          ) : (
            <button
              disabled
              className="w-full bg-gray-300 text-gray-500 font-medium py-2 px-4 rounded-lg cursor-not-allowed"
            >
              Sold Out
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

export default EventCard
