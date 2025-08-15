import { useState, useEffect } from 'react'
import { Search, Calendar, DollarSign } from 'lucide-react'
import EventCard from './EventCard'
import config from '../config/api'

const EventList = () => {
  const [events, setEvents] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    fetchEvents()
  }, [])

  const fetchEvents = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Try to fetch from API first
      const response = await fetch(`${config.apiUrl}/events`)
      if (response.ok) {
        const data = await response.json()
        setEvents(data)
      } else {
        throw new Error('API not available')
      }
    } catch (err) {
      console.log('API not available, showing sample data:', err.message)
      setError('API not available - showing sample data')
      
      // Set sample data for demonstration
      setEvents([
        {
          id: 1,
          title: 'Bitcoin Conference 2024',
          description: 'The biggest Bitcoin conference of the year with world-renowned speakers and networking opportunities.',
          start_time: '2024-06-15T10:00:00Z',
          end_time: '2024-06-15T18:00:00Z',
          price_sats: 50000,
          capacity: 1000,
          is_active: true,
          stream_url: 'https://stream.example.com/bitcoin2024'
        },
        {
          id: 2,
          title: 'Lightning Network Workshop',
          description: 'Learn about Lightning Network development and build your first Lightning application.',
          start_time: '2024-07-01T14:00:00Z',
          end_time: '2024-07-01T16:00:00Z',
          price_sats: 25000,
          capacity: 100,
          is_active: true,
          stream_url: 'https://stream.example.com/lightning-workshop'
        },
        {
          id: 3,
          title: 'UMA Protocol Deep Dive',
          description: 'Explore the Universal Money Address protocol and its applications in modern payment systems.',
          start_time: '2024-08-10T09:00:00Z',
          end_time: '2024-08-10T17:00:00Z',
          price_sats: 75000,
          capacity: 500,
          is_active: true,
          stream_url: 'https://stream.example.com/uma-deep-dive'
        }
      ])
    } finally {
      setLoading(false)
    }
  }

  // Filter events based on search query
  const filteredEvents = events.filter(event =>
    event.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    event.description.toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">Upcoming Events</h1>
        <p className="text-gray-600 max-w-2xl mx-auto">
          Discover amazing virtual events and purchase tickets with Lightning payments
        </p>
      </div>

      {/* API Status Notice */}
      {error && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-yellow-800">
                Demo Mode
              </h3>
              <div className="mt-2 text-sm text-yellow-700">
                <p>Backend API is not available. Showing sample events for demonstration purposes.</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Search Bar */}
      <div className="max-w-md mx-auto">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search events..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
        </div>
      </div>

      {/* Events Grid */}
      {filteredEvents.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-gray-400 mb-4">
            <svg className="mx-auto h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">No events found</h3>
          <p className="text-gray-600">
            {searchQuery ? 'Try adjusting your search terms.' : 'Check back later for upcoming events.'}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredEvents.map((event) => (
            <EventCard key={event.id} event={event} />
          ))}
        </div>
      )}

      {/* Event Count */}
      <div className="text-center text-sm text-gray-500">
        Showing {filteredEvents.length} of {events.length} events
      </div>
    </div>
  )
}

export default EventList
