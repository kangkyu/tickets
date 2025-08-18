import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import config from '../config/api'

const AdminDashboard = () => {
  const { user, token } = useAuth()
  const location = useLocation()
  const [events, setEvents] = useState([])
  const [pendingPayments, setPendingPayments] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [successMessage, setSuccessMessage] = useState('')
  
  // UMA Request state
  const [umaRequest, setUmaRequest] = useState({
    umaAddress: '',
    amountSats: '',
    description: ''
  })
  const [umaRequestLoading, setUmaRequestLoading] = useState(false)
  const [umaRequestError, setUmaRequestError] = useState('')
  const [umaRequestSuccess, setUmaRequestSuccess] = useState('')

  useEffect(() => {
    fetchAdminData()
    
    // Check for success message from navigation state
    if (location.state?.message) {
      setSuccessMessage(location.state.message)
      // Clear the message from navigation state
      window.history.replaceState({}, document.title)
    }
  }, [location.state])

  const fetchAdminData = async () => {
    try {
      setLoading(true)
      
      // Fetch all events
      const eventsResponse = await fetch(`${config.apiUrl}/api/events`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      
      if (eventsResponse.ok) {
        const eventsData = await eventsResponse.json()
        setEvents(eventsData.data || [])
      }

      // Fetch pending payments (admin only)
      const paymentsResponse = await fetch(`${config.apiUrl}/api/admin/payments/pending`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      
      if (paymentsResponse.ok) {
        const paymentsData = await paymentsResponse.json()
        setPendingPayments(paymentsData.data || [])
      }

    } catch (error) {
      console.error('Failed to fetch admin data:', error)
      setError('Failed to load admin data')
    } finally {
      setLoading(false)
    }
  }

  const handleUMARequestChange = (e) => {
    const { name, value } = e.target
    setUmaRequest(prev => ({
      ...prev,
      [name]: value
    }))
  }

  const handleCreateUMARequest = async (e) => {
    e.preventDefault()
    
    try {
      setUmaRequestLoading(true)
      setUmaRequestError('')
      setUmaRequestSuccess('')
      
      const response = await fetch(`${config.apiUrl}/api/admin/uma/requests`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          uma_address: umaRequest.umaAddress,
          amount_sats: parseInt(umaRequest.amountSats),
          description: umaRequest.description
        })
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to create UMA request')
      }
      
      const result = await response.json()
      setUmaRequestSuccess('UMA Request created successfully!')
      
      // Reset form
      setUmaRequest({
        umaAddress: '',
        amountSats: '',
        description: ''
      })
      
      // Clear success message after 5 seconds
      setTimeout(() => {
        setUmaRequestSuccess('')
      }, 5000)
      
    } catch (err) {
      setUmaRequestError(err.message)
    } finally {
      setUmaRequestLoading(false)
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
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Admin Dashboard</h1>
        <p className="text-gray-600">Welcome, {user?.name || user?.email}</p>
      </div>

      {error && (
        <div className="mb-6 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {successMessage && (
        <div className="mb-6 bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
          {successMessage}
        </div>
      )}

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
        <Link
          to="/admin/events/new"
          className="bg-blue-500 hover:bg-blue-600 text-white p-6 rounded-lg shadow-md transition-colors"
        >
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
            </div>
            <div className="ml-4">
              <h3 className="text-lg font-semibold">Add New Event</h3>
              <p className="text-blue-100">Create event with UMA invoices</p>
            </div>
          </div>
        </Link>

        <div className="bg-green-500 text-white p-6 rounded-lg shadow-md">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <div className="ml-4">
              <h3 className="text-lg font-semibold">Total Events</h3>
              <p className="text-2xl font-bold">{events.length}</p>
            </div>
          </div>
        </div>

        <div className="bg-yellow-500 text-white p-6 rounded-lg shadow-md">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <svg className="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1" />
              </svg>
            </div>
            <div className="ml-4">
              <h3 className="text-lg font-semibold">Pending Payments</h3>
              <p className="text-2xl font-bold">{pendingPayments.length}</p>
            </div>
          </div>
        </div>
      </div>

      {/* UMA Request Creation */}
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-900 mb-4">UMA Request Management</h2>
        <div className="bg-white shadow rounded-lg p-6">
          <div className="mb-4">
            <h3 className="text-lg font-medium text-gray-900 mb-2">Create Business UMA Request</h3>
            <p className="text-sm text-gray-600">
              Create UMA Request invoices for business services. According to UMA protocol: "A business or individual creates a one-time invoice using UMA Request for a product or service."
            </p>
          </div>
          
          <form onSubmit={handleCreateUMARequest} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label htmlFor="umaAddress" className="block text-sm font-medium text-gray-700">
                  UMA Address
                </label>
                <input
                  type="text"
                  id="umaAddress"
                  name="umaAddress"
                  value={umaRequest.umaAddress}
                  onChange={handleUMARequestChange}
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                  placeholder="$username@domain.com"
                  required
                />
              </div>
              
              <div>
                <label htmlFor="amountSats" className="block text-sm font-medium text-gray-700">
                  Amount (sats)
                </label>
                <input
                  type="number"
                  id="amountSats"
                  name="amountSats"
                  value={umaRequest.amountSats}
                  onChange={handleUMARequestChange}
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                  placeholder="1000"
                  min="1"
                  required
                />
              </div>
              
              <div>
                <label htmlFor="description" className="block text-sm font-medium text-gray-700">
                  Description
                </label>
                <input
                  type="text"
                  id="description"
                  name="description"
                  value={umaRequest.description}
                  onChange={handleUMARequestChange}
                  className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                  placeholder="Multi-use invoice for services"
                  required
                />
              </div>
            </div>
            
            <div className="flex justify-end">
              <button
                type="submit"
                disabled={umaRequestLoading}
                className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-purple-600 hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-purple-500 disabled:opacity-50"
              >
                {umaRequestLoading ? 'Creating...' : 'Create UMA Request'}
              </button>
            </div>
          </form>
          
          {umaRequestError && (
            <div className="mt-4 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
              {umaRequestError}
            </div>
          )}
          
          {umaRequestSuccess && (
            <div className="mt-4 bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
              {umaRequestSuccess}
            </div>
          )}
        </div>
      </div>

      {/* Events Management */}
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-900 mb-4">Events Management</h2>
        <div className="bg-white shadow overflow-hidden sm:rounded-md">
          {events.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-500">No events created yet.</p>
              <Link
                to="/admin/events/new"
                className="mt-2 inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
              >
                Create your first event
              </Link>
            </div>
          ) : (
            <ul className="divide-y divide-gray-200">
              {events.map((event) => (
                <li key={event.id} className="px-4 py-4 sm:px-6">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <h3 className="text-lg font-medium text-gray-900">{event.title}</h3>
                      <p className="text-sm text-gray-500">{event.description}</p>
                      <div className="mt-2 flex items-center text-sm text-gray-500">
                        <span>Capacity: {event.capacity}</span>
                        <span className="mx-2">•</span>
                        <span>Price: {event.price_sats} sats</span>
                        <span className="mx-2">•</span>
                        <span className={`px-2 py-1 rounded-full text-xs ${
                          event.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {event.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                    </div>
                    <div className="flex space-x-2">
                      <Link
                        to={`/events/${event.id}`}
                        className="inline-flex items-center px-3 py-2 border border-gray-300 text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
                      >
                        View
                      </Link>
                      <Link
                        to={`/admin/events/${event.id}/edit`}
                        className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
                      >
                        Edit
                      </Link>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {/* Pending Payments */}
      {pendingPayments.length > 0 && (
        <div className="mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Pending Payments</h2>
          <div className="bg-white shadow overflow-hidden sm:rounded-md">
            <ul className="divide-y divide-gray-200">
              {pendingPayments.map((payment) => (
                <li key={payment.id} className="px-4 py-4 sm:px-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        Payment ID: {payment.id}
                      </p>
                      <p className="text-sm text-gray-500">
                        Amount: {payment.amount_sats} sats
                      </p>
                      <p className="text-sm text-gray-500">
                        Invoice: {payment.invoice_id}
                      </p>
                    </div>
                    <div className="text-sm text-gray-500">
                      {new Date(payment.created_at).toLocaleDateString()}
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  )
}

export default AdminDashboard