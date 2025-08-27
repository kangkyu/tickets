import axios from 'axios'
import config from '../config/api'

// Create axios instance with default configuration
const api = axios.create({
  baseURL: config.apiUrl,
  timeout: config.requestTimeout,
  headers: {
    'Content-Type': 'application/json',
  },
  // Add CORS-specific options
  withCredentials: true, // Important for CORS with credentials
})

// CORS debugging and fallback logic
const handleCORSIssue = (error, config) => {
  console.warn('🔍 CORS Debug Info:', {
    url: config.url,
    method: config.method,
    origin: window.location.origin,
    target: new URL(config.url).origin,
    error: error.message,
    status: error.response?.status,
    headers: error.response?.headers,
  })

  // Check if it's a CORS issue
  if (error.message.includes('CORS') || error.message.includes('blocked by CORS policy')) {
    console.error('🚫 CORS Issue Detected!')
    console.error('💡 Possible Solutions:')
    console.error('   1. Check if backend CORS middleware is working')
    console.error('   2. Verify allowed origins in backend config')
    console.error('   3. Check if backend is running and accessible')
    console.error('   4. Try using a CORS proxy for development')
    
    // Suggest fallback for development
    if (config.isDevelopment) {
      console.warn('🛠️  Development Fallback: Consider using a CORS proxy')
    }
  }
}

// Request interceptor for adding auth tokens or other headers
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const token = localStorage.getItem('authToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    
    // Add CORS debugging info
    config.isDevelopment = import.meta.env.MODE === 'development'
    
    // Log request details for debugging
    if (config.isDevelopment) {
      console.log('🚀 API Request:', {
        method: config.method?.toUpperCase(),
        url: config.url,
        baseURL: config.baseURL,
        headers: config.headers,
      })
    }
    
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    // Handle CORS issues specifically
    if (error.message.includes('CORS') || error.message.includes('blocked by CORS policy')) {
      handleCORSIssue(error, error.config)
    }
    
    if (error.response) {
      // Server responded with error status
      const { status, data } = error.response
      
      // Log response details for debugging
      if (error.config?.isDevelopment) {
        console.log('📡 API Response Error:', {
          status,
          url: error.config.url,
          method: error.config.method,
          headers: error.response.headers,
          data,
        })
      }
      
      switch (status) {
        case 401:
          // Unauthorized - clear token and redirect to login
          localStorage.removeItem('authToken')
          break
        case 403:
          // Forbidden
          console.error('Access forbidden:', data.message)
          break
        case 404:
          // Not found
          console.error('Resource not found:', data.message)
          break
        case 422:
          // Validation error
          console.error('Validation error:', data.errors)
          break
        case 500:
          // Server error
          console.error('Server error:', data.message)
          break
        default:
          console.error(`HTTP ${status}:`, data.message)
      }
      
      return Promise.reject({
        status,
        message: data.message || 'An error occurred',
        errors: data.errors || {},
      })
    } else if (error.request) {
      // Network error - could be CORS related
      console.error('🌐 Network error:', error.message)
      
      // Check if it's a CORS issue
      if (error.message.includes('CORS') || error.message.includes('blocked by CORS policy')) {
        handleCORSIssue(error, error.config)
      }
      
      return Promise.reject({
        status: 0,
        message: 'Network error - please check your connection',
        errors: {},
      })
    } else {
      // Other error
      console.error('❌ Request error:', error.message)
      return Promise.reject({
        status: 0,
        message: error.message || 'An unexpected error occurred',
        errors: {},
      })
    }
  }
)

// Events API
export const eventsAPI = {
  // Get all events with optional filters
  getAll: async (filters = {}) => {
    const params = new URLSearchParams()
    
    if (filters.query) params.append('query', filters.query)
    if (filters.startDate) params.append('startDate', filters.startDate.toISOString())
    if (filters.endDate) params.append('endDate', filters.endDate.toISOString())
    if (filters.maxPrice) params.append('maxPrice', filters.maxPrice)
    if (filters.category) params.append('category', filters.category)
    
    const response = await api.get(`/api/events?${params.toString()}`)
    // Extract the actual events data from the response wrapper
    return response.data || response
  },

  // Get single event by ID
  getById: async (eventId) => {
    const response = await api.get(`/api/events/${eventId}`)
    // Extract the actual event data from the response wrapper
    return response.data || response
  },

  // Search events
  search: async (query, filters = {}) => {
    const params = new URLSearchParams({ query, ...filters })
    return api.get(`/api/events/search?${params.toString()}`)
  },
}

// Tickets API
export const ticketsAPI = {
  // Purchase a ticket
  purchase: async (ticketData) => {
    return api.post('/api/tickets/purchase', ticketData)
  },

  // Get ticket status
  getStatus: async (ticketId) => {
    return api.get(`/api/tickets/${ticketId}/status`)
  },

  // Get user's tickets
  getUserTickets: async (userId) => {
    return api.get(`/api/users/${userId}/tickets`)
  },

  // Validate a ticket
  validate: async (ticketData) => {
    return api.post('/api/tickets/validate', ticketData)
  },

  // Get ticket details
  getById: async (ticketId) => {
    return api.get(`/api/tickets/${ticketId}`)
  },
}

// Payments API
export const paymentsAPI = {
  // Get payment status
  getStatus: async (invoiceId) => {
    return api.get(`/api/payments/${invoiceId}/status`)
  },

  // Create payment invoice
  createInvoice: async (paymentData) => {
    return api.post('/api/payments/create', paymentData)
  },

  // Cancel payment
  cancel: async (invoiceId) => {
    return api.post(`/api/payments/${invoiceId}/cancel`)
  },
}

// User API
export const userAPI = {
  // Get user profile
  getProfile: async () => {
    return api.get('/api/user/profile')
  },

  // Update user profile
  updateProfile: async (profileData) => {
    return api.put('/api/user/profile', profileData)
  },

  // Get user preferences
  getPreferences: async () => {
    return api.get('/api/user/preferences')
  },
}

// Utility functions
export const apiUtils = {
  // Check if user is online
  isOnline: () => navigator.onLine,

  // Test CORS connectivity
  testCORS: async (endpoint = '/health') => {
    try {
      console.log('🧪 Testing CORS connectivity...')
      
      const testUrl = `${config.apiUrl}${endpoint}`
      console.log('📍 Testing URL:', testUrl)
      
      const response = await fetch(testUrl, {
        method: 'GET',
        mode: 'cors',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
      })
      
      console.log('✅ CORS Test Success:', {
        status: response.status,
        statusText: response.statusText,
        headers: Object.fromEntries(response.headers.entries()),
      })
      
      return { success: true, response }
    } catch (error) {
      console.error('❌ CORS Test Failed:', error)
      
      if (error.message.includes('CORS')) {
        console.error('🚫 CORS Issue Details:')
        console.error('   - Origin:', window.location.origin)
        console.error('   - Target:', config.apiUrl)
        console.error('   - Error:', error.message)
      }
      
      return { success: false, error }
    }
  },

  // Retry function with exponential backoff
  retry: async (fn, maxRetries = config.maxRetries, delay = 1000) => {
    try {
      return await fn()
    } catch (error) {
      if (maxRetries <= 0) throw error
      
      await new Promise(resolve => setTimeout(resolve, delay))
      return apiUtils.retry(fn, maxRetries - 1, delay * 2)
    }
  },

  // Poll function for payment status
  poll: async (fn, interval = config.pollInterval, timeout = config.paymentTimeout) => {
    const startTime = Date.now()
    
    while (Date.now() - startTime < timeout) {
      try {
        const result = await fn()
        if (result.status === 'paid' || result.status === 'expired') {
          return result
        }
      } catch (error) {
        console.warn('Polling error:', error.message)
      }
      
      await new Promise(resolve => setTimeout(resolve, interval))
    }
    
    throw new Error('Polling timeout exceeded')
  },
}

export default api

