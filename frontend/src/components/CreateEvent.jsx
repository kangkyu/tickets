import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import config from '../config/api'

const CreateEvent = () => {
  const { token } = useAuth()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    startTime: '',
    endTime: '',
    capacity: '',
    priceSats: '',
    streamUrl: ''
  })

  const handleInputChange = (e) => {
    const { name, value } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: value
    }))
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      // Validate form data
      if (!formData.title || !formData.startTime || !formData.endTime || !formData.capacity || !formData.priceSats) {
        throw new Error('Please fill in all required fields')
      }

      const startTime = new Date(formData.startTime)
      const endTime = new Date(formData.endTime)
      
      if (startTime >= endTime) {
        throw new Error('End time must be after start time')
      }
      
      if (startTime <= new Date()) {
        throw new Error('Start time must be in the future')
      }

      if (parseInt(formData.capacity) <= 0) {
        throw new Error('Capacity must be greater than 0')
      }

      if (parseInt(formData.priceSats) <= 0) {
        throw new Error('Price must be greater than 0')
      }

      const eventData = {
        title: formData.title,
        description: formData.description,
        start_time: startTime.toISOString(),
        end_time: endTime.toISOString(),
        capacity: parseInt(formData.capacity),
        price_sats: parseInt(formData.priceSats),
        stream_url: formData.streamUrl
      }

      const response = await fetch(`${config.apiUrl}/api/admin/events`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(eventData)
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to create event')
      }

      const result = await response.json()
      
      // Redirect to admin dashboard with success message
      navigate('/admin', { 
        state: { 
          message: 'Event created successfully! Users can now purchase tickets using UMA invoices.' 
        } 
      })

    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    navigate('/admin')
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Create New Event</h1>
            <p className="text-gray-600 mt-2">
              Create an event that users can purchase tickets for using UMA invoices
            </p>
          </div>
          <button
            onClick={handleCancel}
            className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
          >
            Cancel
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-6 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {/* Form */}
      <div className="bg-white shadow sm:rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Event Title */}
            <div>
              <label htmlFor="title" className="block text-sm font-medium text-gray-700">
                Event Title *
              </label>
              <input
                type="text"
                name="title"
                id="title"
                required
                value={formData.title}
                onChange={handleInputChange}
                className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                placeholder="Enter event title"
              />
            </div>

            {/* Description */}
            <div>
              <label htmlFor="description" className="block text-sm font-medium text-gray-700">
                Description
              </label>
              <textarea
                name="description"
                id="description"
                rows={3}
                value={formData.description}
                onChange={handleInputChange}
                className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                placeholder="Enter event description"
              />
            </div>

            {/* Date and Time */}
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
              <div>
                <label htmlFor="startTime" className="block text-sm font-medium text-gray-700">
                  Start Time *
                </label>
                <input
                  type="datetime-local"
                  name="startTime"
                  id="startTime"
                  required
                  value={formData.startTime}
                  onChange={handleInputChange}
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                />
              </div>

              <div>
                <label htmlFor="endTime" className="block text-sm font-medium text-gray-700">
                  End Time *
                </label>
                <input
                  type="datetime-local"
                  name="endTime"
                  id="endTime"
                  required
                  value={formData.endTime}
                  onChange={handleInputChange}
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                />
              </div>
            </div>

            {/* Capacity and Price */}
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
              <div>
                <label htmlFor="capacity" className="block text-sm font-medium text-gray-700">
                  Capacity *
                </label>
                <input
                  type="number"
                  name="capacity"
                  id="capacity"
                  required
                  min="1"
                  value={formData.capacity}
                  onChange={handleInputChange}
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                  placeholder="Maximum number of attendees"
                />
              </div>

              <div>
                <label htmlFor="priceSats" className="block text-sm font-medium text-gray-700">
                  Price (Sats) *
                </label>
                <div className="mt-1 relative rounded-md shadow-sm">
                  <input
                    type="number"
                    name="priceSats"
                    id="priceSats"
                    required
                    min="1"
                    value={formData.priceSats}
                    onChange={handleInputChange}
                    className="block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    placeholder="Price in satoshis"
                  />
                  <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
                    <span className="text-gray-500 sm:text-sm">sats</span>
                  </div>
                </div>
                <p className="mt-1 text-sm text-gray-500">
                  Price in satoshis (1 BTC = 100,000,000 sats)
                </p>
              </div>
            </div>

            {/* Stream URL */}
            <div>
              <label htmlFor="streamUrl" className="block text-sm font-medium text-gray-700">
                Stream URL
              </label>
              <input
                type="url"
                name="streamUrl"
                id="streamUrl"
                value={formData.streamUrl}
                onChange={handleInputChange}
                className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                placeholder="https://stream.example.com/event"
              />
              <p className="mt-1 text-sm text-gray-500">
                Optional: URL where users can watch the event stream
              </p>
            </div>

            {/* UMA Invoice Info */}
            <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-blue-800">
                    UMA Invoice Integration
                  </h3>
                  <div className="mt-2 text-sm text-blue-700">
                    <p>
                      When users purchase tickets, the system will automatically generate UMA invoices 
                      for Lightning Network payments. No pre-creation of invoices is needed.
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* Submit Button */}
            <div className="flex justify-end space-x-3">
              <button
                type="button"
                onClick={handleCancel}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Creating...' : 'Create Event'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

export default CreateEvent
