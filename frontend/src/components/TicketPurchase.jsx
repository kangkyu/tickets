import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { ArrowLeft, User, Mail, CreditCard, CheckCircle, AlertCircle } from 'lucide-react'
import { useEvent } from '../hooks/useEvents'
import { useTicketPurchase } from '../hooks/useTickets'
import { purchaseTicketSchema } from '../utils/validators'
import { formatPrice, formatSatsToUSD } from '../utils/formatters'

const TicketPurchase = () => {
  const { eventId } = useParams()
  const navigate = useNavigate()
  const [currentStep, setCurrentStep] = useState(1)
  const [purchaseData, setPurchaseData] = useState(null)
  
  // Get event data
  const { data: event, isLoading: eventLoading, error: eventError } = useEvent(eventId)
  
  // Ticket purchase mutation
  const { mutate: purchaseTicket, isLoading: isPurchasing, error: purchaseError } = useTicketPurchase()

  // Form setup
  const {
    register,
    handleSubmit,
    formState: { errors, isValid },
    watch,
    setValue
  } = useForm({
    resolver: zodResolver(purchaseTicketSchema),
    defaultValues: {
      eventId: parseInt(eventId),
      quantity: 1,
      userName: '',
      userEmail: '',
      umaAddress: ''
    },
    mode: 'onChange'
  })

  const watchedValues = watch()

  // Handle form submission
  const onSubmit = (data) => {
    setPurchaseData(data)
    setCurrentStep(2)
    
    // Submit purchase request
    purchaseTicket(data, {
      onSuccess: (response) => {
        // Navigate to payment status page
        navigate(`/tickets/${response.ticketId}/payment`)
      },
      onError: (error) => {
        console.error('Purchase failed:', error)
        setCurrentStep(1)
      }
    })
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
  const isEventPassed = new Date(event.date) < new Date()
  const isSoldOut = event.availableTickets === 0

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
                    {...register('quantity')}
                    className="input-field w-auto"
                  >
                    {[...Array(Math.min(10, event.availableTickets))].map((_, i) => (
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
                        {...register('userName')}
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
                        {...register('userEmail')}
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
                      <CreditCard className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
                      <input
                        type="text"
                        {...register('umaAddress')}
                        className="input-field pl-10"
                        placeholder="username@domain.com"
                      />
                    </div>
                    {errors.umaAddress && (
                      <p className="mt-1 text-sm text-red-600">{errors.umaAddress.message}</p>
                    )}
                    <p className="mt-1 text-xs text-gray-500">
                      Enter your UMA address for payment processing
                    </p>
                  </div>
                </div>

                {/* Purchase Error */}
                {purchaseError && (
                  <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                    <div className="flex items-center gap-2">
                      <AlertCircle className="w-5 h-5 text-red-500" />
                      <p className="text-red-700 font-medium">Purchase failed</p>
                    </div>
                    <p className="text-red-600 text-sm mt-1">{purchaseError.message}</p>
                  </div>
                )}

                {/* Submit Button */}
                <button
                  type="submit"
                  disabled={!isValid || isPurchasing}
                  className="btn-uma w-full disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isPurchasing ? 'Processing...' : 'Continue to Payment'}
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
                  <img
                    src={event.imageUrl || '/placeholder-event.jpg'}
                    alt={event.title}
                    className="w-16 h-16 object-cover rounded-lg"
                  />
                  <div className="flex-1">
                    <h4 className="font-medium text-gray-900">{event.title}</h4>
                    <p className="text-sm text-gray-600">{event.date}</p>
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
                    <span>{formatPrice(event.price)}</span>
                  </div>
                  <div className="border-t pt-3 flex justify-between font-semibold">
                    <span>Total:</span>
                    <span className="text-uma-600">
                      {formatPrice((watchedValues.quantity || 1) * event.price)}
                    </span>
                  </div>
                  <div className="text-xs text-gray-500 text-center">
                    ≈ ${formatSatsToUSD((watchedValues.quantity || 1) * event.price)}
                  </div>
                </div>

                {/* Payment Info */}
                <div className="bg-gray-50 p-3 rounded-lg">
                  <p className="text-xs text-gray-500 mb-2">Payment Method</p>
                  <div className="flex items-center gap-2">
                    <div className="w-6 h-6 bg-uma-600 rounded flex items-center justify-center">
                      <span className="text-white text-xs font-bold">⚡</span>
                    </div>
                    <span className="text-sm font-medium text-gray-900">Lightning Network</span>
                  </div>
                  <p className="text-xs text-gray-500 mt-1">
                    Instant payment with Bitcoin Lightning
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
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Processing Your Order</h2>
          <p className="text-gray-600">
            Please wait while we generate your Lightning invoice...
          </p>
        </div>
      )}
    </div>
  )
}

export default TicketPurchase
