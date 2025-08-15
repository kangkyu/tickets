const config = {
  apiUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  isDevelopment: import.meta.env.MODE === 'development',
  isProduction: import.meta.env.MODE === 'production',
  pollInterval: 10000, // Payment status polling interval (10 seconds)
  paymentTimeout: 300000, // 5 minutes in milliseconds
  maxRetries: 3,
  requestTimeout: 10000, // 10 seconds
}

export default config
