import { useState, useMemo } from 'react'
import { Search, Filter, Calendar, MapPin, DollarSign } from 'lucide-react'
import { useEvents } from '../hooks/useEvents'
import EventCard from './EventCard'

const EventList = () => {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('')
  const [priceRange, setPriceRange] = useState('')
  const [sortBy, setSortBy] = useState('date')

  // Get events data
  const { data: events = [], isLoading, error } = useEvents()

  // Filter and sort events
  const filteredEvents = useMemo(() => {
    let filtered = events

    // Search filter
    if (searchQuery) {
      filtered = filtered.filter(event =>
        event.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        event.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
        event.location?.toLowerCase().includes(searchQuery.toLowerCase())
      )
    }

    // Category filter
    if (selectedCategory) {
      filtered = filtered.filter(event => event.category === selectedCategory)
    }

    // Price filter
    if (priceRange) {
      const [min, max] = priceRange.split('-').map(Number)
      filtered = filtered.filter(event => {
        if (max) {
          return event.price >= min && event.price <= max
        }
        return event.price >= min
      })
    }

    // Sort events
    filtered.sort((a, b) => {
      switch (sortBy) {
        case 'date':
          return new Date(a.date) - new Date(b.date)
        case 'price-low':
          return a.price - b.price
        case 'price-high':
          return b.price - a.price
        case 'name':
          return a.title.localeCompare(b.title)
        default:
          return 0
      }
    })

    return filtered
  }, [events, searchQuery, selectedCategory, priceRange, sortBy])

  // Get unique categories
  const categories = useMemo(() => {
    const uniqueCategories = [...new Set(events.map(event => event.category).filter(Boolean))]
    return uniqueCategories.sort()
  }, [events])

  // Loading skeleton
  if (isLoading) {
    return (
      <div className="space-y-8">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-900 mb-4">Discover Events</h1>
          <p className="text-gray-600">Find amazing events and purchase tickets with Lightning payments</p>
        </div>
        
        {/* Search and Filter Skeleton */}
        <div className="space-y-4">
          <div className="loading-skeleton h-12 w-full rounded-lg"></div>
          <div className="flex gap-4">
            <div className="loading-skeleton h-10 w-32 rounded-lg"></div>
            <div className="loading-skeleton h-10 w-32 rounded-lg"></div>
            <div className="loading-skeleton h-10 w-32 rounded-lg"></div>
          </div>
        </div>

        {/* Events Grid Skeleton */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="card">
              <div className="loading-skeleton h-48 w-full rounded-lg mb-4"></div>
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
    )
  }

  // Error state
  if (error) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">⚠️</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Something went wrong</h2>
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
        <h1 className="text-3xl font-bold text-gray-900 mb-4">Discover Events</h1>
        <p className="text-gray-600">Find amazing events and purchase tickets with Lightning payments</p>
      </div>

      {/* Search and Filters */}
      <div className="space-y-4">
        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
          <input
            type="text"
            placeholder="Search events, locations, or categories..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="input-field pl-10"
          />
        </div>

        {/* Filter Controls */}
        <div className="flex flex-wrap gap-4">
          {/* Category Filter */}
          <select
            value={selectedCategory}
            onChange={(e) => setSelectedCategory(e.target.value)}
            className="input-field w-auto min-w-[140px]"
          >
            <option value="">All Categories</option>
            {categories.map(category => (
              <option key={category} value={category}>{category}</option>
            ))}
          </select>

          {/* Price Range Filter */}
          <select
            value={priceRange}
            onChange={(e) => setPriceRange(e.target.value)}
            className="input-field w-auto min-w-[140px]"
          >
            <option value="">Any Price</option>
            <option value="0-10000">Under 10k sats</option>
            <option value="10000-50000">10k - 50k sats</option>
            <option value="50000-100000">50k - 100k sats</option>
            <option value="100000-">Over 100k sats</option>
          </select>

          {/* Sort By */}
          <select
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value)}
            className="input-field w-auto min-w-[140px]"
          >
            <option value="date">Sort by Date</option>
            <option value="price-low">Price: Low to High</option>
            <option value="price-high">Price: High to Low</option>
            <option value="name">Sort by Name</option>
          </select>
        </div>
      </div>

      {/* Results Count */}
      <div className="flex items-center justify-between">
        <p className="text-gray-600">
          {filteredEvents.length} event{filteredEvents.length !== 1 ? 's' : ''} found
        </p>
        
        {searchQuery || selectedCategory || priceRange ? (
          <button
            onClick={() => {
              setSearchQuery('')
              setSelectedCategory('')
              setPriceRange('')
              setSortBy('date')
            }}
            className="text-uma-600 hover:text-uma-700 text-sm font-medium"
          >
            Clear all filters
          </button>
        ) : null}
      </div>

      {/* Events Grid */}
      {filteredEvents.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredEvents.map(event => (
            <EventCard key={event.id} event={event} />
          ))}
        </div>
      ) : (
        <div className="text-center py-12">
          <div className="text-gray-400 text-6xl mb-4">🔍</div>
          <h3 className="text-xl font-semibold text-gray-900 mb-2">No events found</h3>
          <p className="text-gray-600">
            Try adjusting your search criteria or browse all available events.
          </p>
          {(searchQuery || selectedCategory || priceRange) && (
            <button
              onClick={() => {
                setSearchQuery('')
                setSelectedCategory('')
                setPriceRange('')
                setSortBy('date')
              }}
              className="btn-primary mt-4"
            >
              Clear Filters
            </button>
          )}
        </div>
      )}
    </div>
  )
}

export default EventList
