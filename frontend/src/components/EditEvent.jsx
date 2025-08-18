import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import config from '../config/api'

const EditEvent = () => {
  const { token } = useAuth()
  const navigate = useNavigate()
  const { eventId } = useParams()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    startTime: '',
    endTime: '',
    capacity: '',
    priceSats: '',
    streamUrl: '',
    isActive: true
  })

  // UMA Request invoice state
  const [umaInvoice, setUmaInvoice] = useState({
    id: '',
    bolt11: '',
    address: '',
    exists: false
  })
  const [umaLoading, setUmaLoading] = useState(false)
  const [umaError, setUmaError] = useState('')
  const [umaSuccess, setUmaSuccess] = useState('')

  useEffect(() => {
    fetchEvent()
  }, [eventId])

  const fetchEvent = async () => {
    try {
      const response = await fetch(`${config.apiUrl}/api/events/${eventId}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (!response.ok) {
        throw new Error('Failed to fetch event')
      }

      const result = await response.json()
      const event = result.data

      // Convert ISO strings to datetime-local format
      const startTime = new Date(event.start_time)
      const endTime = new Date(event.end_time)
      
      setFormData({
        title: event.title,
        description: event.description || '',
        startTime: startTime.toISOString().slice(0, 16), // Format for datetime-local
        endTime: endTime.toISOString().slice(0, 16),
        capacity: event.capacity.toString(),
        priceSats: event.price_sats.toString(),
        streamUrl: event.stream_url || '',
        isActive: event.is_active
      })

      // Set UMA Request invoice information
      setUmaInvoice({
        id: event.uma_request_invoice?.id || '',
        bolt11: event.uma_request_invoice?.bolt11 || '',
        address: event.uma_request_invoice?.uma_address || '',
        exists: !!(event.uma_request_invoice && event.uma_request_invoice.id)
      })

    } catch (err) {
      setError('Failed to load event: ' + err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }))
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setSaving(true)
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
        stream_url: formData.streamUrl,
        is_active: formData.isActive
      }

      const response = await fetch(`${config.apiUrl}/api/admin/events/${eventId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(eventData)
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to update event')
      }

      // Redirect to admin dashboard with success message
      navigate('/admin', { 
        state: { 
          message: 'Event updated successfully!' 
        } 
      })

    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = () => {
    navigate('/admin')
  }

  const handleCreateUMAInvoice = async () => {
    setUmaLoading(true)
    setUmaError('')
    setUmaSuccess('')

    try {
      const response = await fetch(`${config.apiUrl}/api/admin/events/${eventId}/uma-invoice`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to create UMA Request invoice')
      }

      const result = await response.json()
      const invoice = result.data.invoice

      setUmaInvoice({
        id: invoice.id,
        bolt11: invoice.bolt11,
        address: result.data.uma_address,
        exists: true
      })

      setUmaSuccess('UMA Request invoice created successfully!')
      
      // Clear success message after 5 seconds
      setTimeout(() => {
        setUmaSuccess('')
      }, 5000)

    } catch (err) {
      setUmaError(err.message)
    } finally {
      setUmaLoading(false)
    }
  }

  const handleUpdateUMAInvoice = async () => {
    setUmaLoading(true)
    setUmaError('')
    setUmaSuccess('')

    try {
      // Update the event first to trigger UMA invoice regeneration
      const eventData = {
        price_sats: parseInt(formData.priceSats)
      }

      const response = await fetch(`${config.apiUrl}/api/admin/events/${eventId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(eventData)
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to update event')
      }

      // Fetch the updated event to get new UMA invoice
      const updatedResponse = await fetch(`${config.apiUrl}/api/events/${eventId}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (updatedResponse.ok) {
        const result = await updatedResponse.json()
        const event = result.data

        setUmaInvoice({
          id: event.uma_request_invoice?.id || '',
          bolt11: event.uma_request_invoice?.bolt11 || '',
          address: event.uma_request_invoice?.uma_address || '',
          exists: !!(event.uma_request_invoice && event.uma_request_invoice.id)
        })

        setUmaSuccess('UMA Request invoice updated successfully!')
        
        // Clear success message after 5 seconds
        setTimeout(() => {
          setUmaSuccess('')
        }, 5000)
      }

    } catch (err) {
      setUmaError(err.message)
    } finally {
      setUmaLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-500"></div>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Edit Event</h1>
            <p className="text-gray-600 mt-2">
              Update event details and settings
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

            {/* Active Status */}
            <div className="flex items-center">
              <input
                type="checkbox"
                name="isActive"
                id="isActive"
                checked={formData.isActive}
                onChange={handleInputChange}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="isActive" className="ml-2 block text-sm text-gray-900">
                Event is active (users can purchase tickets)
              </label>
            </div>

            {/* UMA Request Invoice Management */}
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
              <h3 className="text-lg font-semibold text-blue-800 mb-3">
                UMA Request Invoice Management
              </h3>
              
              {formData.priceSats > 0 ? (
                // Paid Event
                umaInvoice.exists ? (
                  <div className="space-y-3">
                    <div className="flex items-center text-green-700">
                      <svg className="h-5 w-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                      </svg>
                      <span className="font-medium">Active UMA Request Invoice</span>
                    </div>
                    <div className="bg-white rounded-md p-3 space-y-2 text-sm">
                      <div><strong>Invoice ID:</strong> {umaInvoice.id}</div>
                      <div><strong>Bolt11:</strong> <code className="bg-gray-100 px-2 py-1 rounded text-xs">{umaInvoice.bolt11}</code></div>
                      <div><strong>UMA Address:</strong> <code className="bg-gray-100 px-2 py-1 rounded text-xs">{umaInvoice.address}</code></div>
                    </div>
                    <button
                      onClick={handleUpdateUMAInvoice}
                      className="bg-yellow-600 hover:bg-yellow-700 text-white px-4 py-2 rounded-md text-sm font-medium"
                    >
                      Update Invoice
                    </button>
                  </div>
                ) : (
                  <div className="space-y-3">
                    <div className="flex items-center text-red-700">
                      <svg className="h-5 w-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                      </svg>
                      <span className="font-medium">Missing UMA Request Invoice</span>
                    </div>
                    <p className="text-red-600 text-sm">
                      This paid event needs a UMA Request invoice for ticket sales. Create one now.
                    </p>
                    <button
                      onClick={handleCreateUMAInvoice}
                      className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md text-sm font-medium"
                    >
                      Create UMA Request Invoice
                    </button>
                  </div>
                )
              ) : (
                // Free Event
                <div className="space-y-3">
                  <div className="flex items-center text-blue-700">
                    <svg className="h-5 w-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                    </svg>
                    <span className="font-medium">Free Event - No UMA Invoice Needed</span>
                  </div>
                  <p className="text-blue-600 text-sm">
                    This is a free event (price = 0). Users can RSVP by getting free tickets without payment. 
                    No UMA Request invoice is required.
                  </p>
                  {umaInvoice.exists && (
                    <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3">
                      <p className="text-yellow-800 text-sm">
                        <strong>Note:</strong> This event has a UMA Request invoice but is set to free. 
                        Consider removing the invoice or setting a price if you want paid tickets.
                      </p>
                    </div>
                  )}
                </div>
              )}

              {umaError && (
                <div className="mt-4 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                  {umaError}
                </div>
              )}

              {umaSuccess && (
                <div className="mt-4 bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
                  {umaSuccess}
                </div>
              )}

              <div className="flex space-x-3 mt-4">
                {!umaInvoice.exists ? (
                  <button
                    type="button"
                    onClick={handleCreateUMAInvoice}
                    disabled={umaLoading}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                  >
                    {umaLoading ? 'Creating...' : 'Create UMA Request Invoice'}
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={handleUpdateUMAInvoice}
                    disabled={umaLoading}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50"
                  >
                    {umaLoading ? 'Updating...' : 'Update UMA Request Invoice'}
                  </button>
                )}
              </div>

              <p className="mt-2 text-sm text-gray-600">
                UMA Request invoices allow users to purchase tickets using Lightning Network payments. 
                Each ticket becomes a "product" that users can buy using the pre-created invoice.
              </p>
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
                disabled={saving}
                className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

export default EditEvent
