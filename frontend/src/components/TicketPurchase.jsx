import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { ArrowLeft, User, Mail, Zap, CheckCircle, AlertCircle, Wallet } from 'lucide-react'
import { UmaConnectButton } from '@uma-sdk/uma-auth-client'
import { useEvent } from '../hooks/useEvents'
import { useAuth } from '../contexts/AuthContext'
import { formatPrice, formatSatsToUSD } from '../utils/formatters'
import config from '../config/api'

const UMA_AUTH_APP_IDENTITY_PUBKEY = import.meta.env.VITE_UMA_AUTH_APP_IDENTITY_PUBKEY || ''
const UMA_AUTH_NOSTR_RELAY = import.meta.env.VITE_UMA_AUTH_NOSTR_RELAY || 'wss://nos.lol'
const UMA_AUTH_REDIRECT_URI = `${window.location.origin}/oauth/callback`

const TicketPurchase = () => {
  const { eventId } = useParams()
  const navigate = useNavigate()
  const [currentStep, setCurrentStep] = useState(1)
  const [isCreatingPayment, setIsCreatingPayment] = useState(false)
  const [paymentError, setPaymentError] = useState(null)
  const [walletConnected, setWalletConnected] = useState(false)
  const [walletError, setWalletError] = useState(null)

  // Check if user already has a wallet connection stored
  const { token } = useAuth()
  const checkWalletConnection = useCallback(async () => {
    if (!token) return
    try {
      const response = await fetch(`${config.apiUrl}/api/users/me/nwc-connection`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (response.ok) {
        setWalletConnected(true)
      }
    } catch (error) {
      // No connection stored, that's fine
    }
  }, [token])

  useEffect(() => {
    checkWalletConnection()
  }, [checkWalletConnection])

  // Get event data
  const { data: event, isLoading: eventLoading, error: eventError } = useEvent(eventId)
  
  // Get authenticated user
  const { user } = useAuth()

  // Form setup
  const {
    register,
    handleSubmit,
    formState: { errors, isValid },
    watch,
    setValue,
    trigger
  } = useForm({
    defaultValues: {
      eventId: parseInt(eventId),
      quantity: 1,
      userName: user?.name || '',
      userEmail: user?.email || '',
      umaAddress: ''
    },
    mode: 'onChange'
  })

  const watchedValues = watch()

  // UMA address validation
  const validateUMAAddress = (address) => {
    if (!address) return { isValid: false, error: 'UMA address is required' }
    
    // UMA addresses follow the format: $username@domain.com
    const umaRegex = /^\$[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
    
    if (!umaRegex.test(address)) {
      return { 
        isValid: false, 
        error: 'Invalid UMA address format. Use format: $username@domain.com' 
      }
    }
    
    return { isValid: true, error: null }
  }

  // Handle UMA address validation
  const handleUMAAddressChange = async (e) => {
    const address = e.target.value
    setValue('umaAddress', address)
    
    // Trigger validation immediately
    const validation = validateUMAAddress(address)
    if (validation.isValid) {
      // Clear any existing errors
      setValue('umaAddress', address, { shouldValidate: true, shouldDirty: true })
    } else {
      // Set the error
      setValue('umaAddress', address, { shouldValidate: true, shouldDirty: true })
    }
    
    // Trigger form validation
    await trigger('umaAddress')
  }

  // Check if form is valid for submission
  const isFormValid = () => {
    const umaAddress = watchedValues.umaAddress
    const userName = watchedValues.userName
    const userEmail = watchedValues.userEmail
    
    const umaValid = validateUMAAddress(umaAddress).isValid
    
    // Debug logging
    console.log('Form validation state:', {
      umaAddress,
      userName,
      userEmail,
      umaValid,
      allValid: umaAddress && userName && userEmail && umaValid
    })
    
    return umaAddress && userName && userEmail && umaValid
  }

  // Handle form submission
  const onSubmit = async (data) => {
    setIsCreatingPayment(true)
    setPaymentError(null)
    setCurrentStep(2)
    
    try {
      // Create ticket purchase with UMA payment
      const response = await fetch(`${config.apiUrl}/api/tickets/purchase`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`
        },
        body: JSON.stringify({
          event_id: data.eventId,
          user_id: user.id,
          uma_address: data.umaAddress
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to create ticket purchase')
      }

      const result = await response.json()
      
      // Navigate to ticket list with success message
      // Handle both paid events (with UMA Request) and free events
      const umaRequest = result.data.uma_request || null
      const isFreeEvent = !result.data.payment_required
      
      navigate(`/tickets`, { 
        state: { 
          successMessage: isFreeEvent 
            ? 'Free ticket created successfully!' 
            : 'Ticket purchase initiated! Please complete your payment.',
          ticketId: result.data.ticket.id,
          invoiceId: umaRequest?.invoice_id || null,
          bolt11: umaRequest?.bolt11 || null,
          amountSats: umaRequest?.amount_sats || 0,
          umaAddress: data.umaAddress,
          ticketData: data,
          isFreeEvent: isFreeEvent
        }
      })
      
    } catch (error) {
      console.error('Ticket purchase failed:', error)
      setPaymentError(error.message)
      setCurrentStep(1)
    } finally {
      setIsCreatingPayment(false)
    }
  }

  // Loading state
  if (eventLoading) {
    return (
      <div className="text-center py-12">
        <div className="loading-skeleton w-16 h-16 rounded-full mx-auto mb-4"></div>
        <div className="loading-skeleton h-6 w-48 mx-auto mb-2"></div>
        <div className="loading-skeleton h-4 w-64 mx-auto"></div>
      </div>
    )
  }

  // Error state
  if (eventError || !event) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">⚠️</div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Event not found</h2>
        <p className="text-gray-600 mb-4">
          {eventError?.message || 'Unable to load event information'}
        </p>
        <button onClick={() => navigate(-1)} className="btn-primary">
          Go Back
        </button>
      </div>
    )
  }

  // Check if event is available
  const isEventPassed = new Date(event.start_time) < new Date()
  const isSoldOut = event.capacity === 0

  if (isEventPassed || isSoldOut) {
    return (
      <div className="text-center py-12">
        <div className="text-red-500 text-6xl mb-4">
          {isEventPassed ? '📅' : '🚫'}
        </div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">
          {isEventPassed ? 'Event has passed' : 'Tickets not available'}
        </h2>
        <p className="text-gray-600 mb-4">
          {isEventPassed 
            ? 'This event has already taken place.'
            : 'All tickets for this event have been sold.'
          }
        </p>
        <button onClick={() => navigate(-1)} className="btn-primary">
          Go Back
        </button>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto space-y-8">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate(-1)}
          className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Purchase Tickets</h1>
          <p className="text-gray-600">{event.title}</p>
        </div>
      </div>

      {/* Progress Steps */}
      <div className="flex items-center justify-center">
        <div className="flex items-center space-x-4">
          <div className={`flex items-center justify-center w-10 h-10 rounded-full border-2 ${
            currentStep >= 1 ? 'bg-uma-600 border-uma-600 text-white' : 'border-gray-300 text-gray-400'
          }`}>
            {currentStep > 1 ? <CheckCircle className="w-5 h-5" /> : '1'}
          </div>
          <div className={`w-16 h-0.5 ${
            currentStep >= 2 ? 'bg-uma-600' : 'bg-gray-300'
          }`}></div>
          <div className={`flex items-center justify-center w-10 h-10 rounded-full border-2 ${
            currentStep >= 2 ? 'bg-uma-600 border-uma-600 text-white' : 'border-gray-300 text-gray-400'
          }`}>
            {currentStep > 2 ? <CheckCircle className="w-5 h-5" /> : '2'}
          </div>
        </div>
      </div>

      {/* Step 1: Purchase Form */}
      {currentStep === 1 && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Form */}
          <div className="lg:col-span-2">
            <div className="card">
              <h2 className="text-xl font-semibold text-gray-900 mb-6">Ticket Information</h2>
              
              <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                {/* Quantity */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Number of Tickets
                  </label>
                  <select
                    {...register('quantity', { required: 'Quantity is required' })}
                    className="input-field w-auto"
                  >
                    {[...Array(Math.min(10, event.capacity))].map((_, i) => (
                      <option key={i + 1} value={i + 1}>
                        {i + 1} ticket{i !== 0 ? 's' : ''}
                      </option>
                    ))}
                  </select>
                  {errors.quantity && (
                    <p className="mt-1 text-sm text-red-600">{errors.quantity.message}</p>
                  )}
                </div>

                {/* User Information */}
                <div className="space-y-4">
                  <h3 className="text-lg font-medium text-gray-900">Personal Information</h3>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Full Name
                    </label>
                    <div className="relative">
                      <User className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
                      <input
                        type="text"
                        {...register('userName', { required: 'Full name is required' })}
                        className="input-field pl-10"
                        placeholder="Enter your full name"
                      />
                    </div>
                    {errors.userName && (
                      <p className="mt-1 text-sm text-red-600">{errors.userName.message}</p>
                    )}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Email Address
                    </label>
                    <div className="relative">
                      <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
                      <input
                        type="email"
                        {...register('userEmail', { 
                          required: 'Email is required',
                          pattern: {
                            value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                            message: 'Invalid email address'
                          }
                        })}
                        className="input-field pl-10"
                        placeholder="Enter your email address"
                      />
                    </div>
                    {errors.userEmail && (
                      <p className="mt-1 text-sm text-red-600">{errors.userEmail.message}</p>
                    )}
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      UMA Address
                    </label>
                    <div className="relative">
                      <Zap className="absolute left-3 top-1/2 transform -translate-y-1/2 text-uma-600 w-5 h-5" />
                      <input
                        type="text"
                        {...register('umaAddress', { 
                          required: 'UMA address is required',
                          validate: (value) => {
                            const validation = validateUMAAddress(value)
                            return validation.isValid || validation.error
                          }
                        })}
                        onChange={handleUMAAddressChange}
                        className={`input-field pl-10 ${
                          watchedValues.umaAddress && validateUMAAddress(watchedValues.umaAddress).isValid
                            ? 'border-green-500 focus:border-green-500 focus:ring-green-500'
                            : watchedValues.umaAddress && !validateUMAAddress(watchedValues.umaAddress).isValid
                            ? 'border-red-500 focus:border-red-500 focus:ring-red-500'
                            : 'border-uma-200 focus:border-uma-500 focus:ring-uma-500'
                        }`}
                        placeholder="$username@domain.com"
                      />
                      {watchedValues.umaAddress && validateUMAAddress(watchedValues.umaAddress).isValid && (
                        <CheckCircle className="absolute right-3 top-1/2 transform -translate-y-1/2 text-green-500 w-5 h-5" />
                      )}
                    </div>
                    {errors.umaAddress && (
                      <p className="mt-1 text-sm text-red-600">{errors.umaAddress.message}</p>
                    )}
                    {watchedValues.umaAddress && validateUMAAddress(watchedValues.umaAddress).isValid && (
                      <p className="mt-1 text-sm text-green-600">✓ Valid UMA address</p>
                    )}
                    <p className="mt-1 text-xs text-gray-500">
                      Enter your UMA address for Lightning Network payment processing
                    </p>
                  </div>
                </div>

                {/* Wallet Connect */}
                {event.price_sats > 0 && (
                  <div className="space-y-4">
                    <h3 className="text-lg font-medium text-gray-900">Connect Wallet</h3>
                    <p className="text-sm text-gray-600">
                      Connect your UMA wallet to enable automatic payment when purchasing tickets.
                    </p>

                    {walletConnected ? (
                      <div className="flex items-center gap-2 p-3 bg-green-50 border border-green-200 rounded-lg">
                        <CheckCircle className="w-5 h-5 text-green-500" />
                        <span className="text-green-700 font-medium">Wallet Connected</span>
                      </div>
                    ) : (
                      <div className="space-y-3">
                        {UMA_AUTH_APP_IDENTITY_PUBKEY ? (
                          <div onClick={() => sessionStorage.setItem('oauth_return_to', window.location.pathname)}>
                            <UmaConnectButton
                              app-identity-pubkey={UMA_AUTH_APP_IDENTITY_PUBKEY}
                              nostr-relay={UMA_AUTH_NOSTR_RELAY}
                              redirect-uri={UMA_AUTH_REDIRECT_URI}
                              required-commands={['pay_invoice']}
                              budget-amount="100000"
                              budget-currency="SAT"
                              budget-period="monthly"
                            />
                          </div>
                        ) : (
                          <div className="flex items-center gap-2 p-3 bg-gray-50 border border-gray-200 rounded-lg">
                            <Wallet className="w-5 h-5 text-gray-400" />
                            <span className="text-gray-500 text-sm">
                              Wallet connect not configured. Payment will be requested via UMA.
                            </span>
                          </div>
                        )}
                        {walletError && (
                          <p className="text-sm text-red-600">{walletError}</p>
                        )}
                      </div>
                    )}
                  </div>
                )}

                {/* Payment Error */}
                {paymentError && (
                  <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                    <div className="flex items-center gap-2">
                      <AlertCircle className="w-5 h-5 text-red-500" />
                      <p className="text-red-700 font-medium">Purchase failed</p>
                    </div>
                    <p className="text-red-600 text-sm mt-1">{paymentError}</p>
                  </div>
                )}

                {/* Submit Button */}
                <button
                  type="submit"
                  disabled={!isFormValid() || isCreatingPayment}
                  className="btn-uma w-full disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isCreatingPayment ? 'Creating Payment...' : 'Continue to Payment'}
                </button>
              </form>
            </div>
          </div>

          {/* Order Summary */}
          <div className="lg:col-span-1">
            <div className="card sticky top-8">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Order Summary</h3>
              
              <div className="space-y-4">
                {/* Event Info */}
                <div className="flex items-start gap-3">
                  <div className="w-16 h-16 bg-gradient-to-br from-uma-500 to-uma-700 rounded-lg flex items-center justify-center">
                    <Zap className="w-6 h-6 text-white opacity-80" />
                  </div>
                  <div className="flex-1">
                    <h4 className="font-medium text-gray-900">{event.title}</h4>
                    <p className="text-sm text-gray-600">{event.start_time}</p>
                  </div>
                </div>

                {/* Ticket Details */}
                <div className="border-t pt-4 space-y-3">
                  <div className="flex justify-between text-sm">
                    <span>Quantity:</span>
                    <span>{watchedValues.quantity || 1}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span>Price per ticket:</span>
                    <span>{formatPrice(event.price_sats)}</span>
                  </div>
                  <div className="border-t pt-3 flex justify-between font-semibold">
                    <span>Total:</span>
                    <span className="text-uma-600">
                      {formatPrice((watchedValues.quantity || 1) * event.price_sats)}
                    </span>
                  </div>
                  <div className="text-xs text-gray-500 text-center">
                    ≈ ${formatSatsToUSD((watchedValues.quantity || 1) * event.price_sats)}
                  </div>
                </div>

                {/* Payment Info */}
                <div className="bg-uma-50 p-3 rounded-lg border border-uma-200">
                  <p className="text-xs text-uma-700 mb-2 font-medium">Payment Method</p>
                  <div className="flex items-center gap-2">
                    <div className="w-6 h-6 bg-uma-600 rounded flex items-center justify-center">
                      <span className="text-white text-xs font-bold">⚡</span>
                    </div>
                    <span className="text-sm font-medium text-uma-900">Lightning Network</span>
                  </div>
                  <p className="text-xs text-uma-600 mt-1">
                    Instant payment with Bitcoin Lightning via UMA
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Step 2: Processing */}
      {currentStep === 2 && (
        <div className="text-center py-12">
          <div className="loading-skeleton w-16 h-16 rounded-full mx-auto mb-4 animate-spin"></div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Creating UMA Payment</h2>
          <p className="text-gray-600">
            Please wait while we generate your Lightning invoice...
          </p>
        </div>
      )}
    </div>
  )
}

export default TicketPurchase
