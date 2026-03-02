import { useEffect, useCallback, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useOAuth } from '@uma-sdk/uma-auth-client'
import { useAuth } from '../contexts/AuthContext'
import config from '../config/api'

const UMA_AUTH_APP_IDENTITY_PUBKEY = import.meta.env.VITE_UMA_AUTH_APP_IDENTITY_PUBKEY || ''
const UMA_AUTH_NOSTR_RELAY = import.meta.env.VITE_UMA_AUTH_NOSTR_RELAY || 'wss://nos.lol'
const UMA_AUTH_REDIRECT_URI = `${window.location.origin}/oauth/callback`

const OAuthCallback = () => {
  const { nwcConnectionUri, oAuthTokenExchange, setAuthConfig } = useOAuth()
  const { token } = useAuth()
  const navigate = useNavigate()
  const exchangeStarted = useRef(false)
  const [error, setError] = useState(null)
  const [status, setStatus] = useState('Exchanging authorization code...')

  // Exchange the authorization code for NWC connection URI
  useEffect(() => {
    if (exchangeStarted.current || nwcConnectionUri) return
    exchangeStarted.current = true

    // Restore auth config (not persisted across page navigations)
    setAuthConfig({
      identityNpub: UMA_AUTH_APP_IDENTITY_PUBKEY,
      identityRelayUrl: UMA_AUTH_NOSTR_RELAY,
      redirectUri: UMA_AUTH_REDIRECT_URI,
    })

    console.log('[OAuthCallback] Starting token exchange, URL params:', window.location.search)
    oAuthTokenExchange()
      .then((result) => {
        console.log('[OAuthCallback] Token exchange succeeded')
        setStatus('Token exchange complete, storing connection...')
      })
      .catch((err) => {
        console.error('[OAuthCallback] Token exchange failed:', err)
        setError(`Token exchange failed: ${err.message || err}`)
      })
  }, [nwcConnectionUri, oAuthTokenExchange, setAuthConfig])

  const storeAndRedirect = useCallback(async (uri) => {
    if (!token) {
      console.warn('[OAuthCallback] No auth token available, cannot store NWC connection')
      setError('Not authenticated. Please log in and try again.')
      return
    }

    console.log('[OAuthCallback] Storing NWC connection')
    setStatus('Saving wallet connection...')

    try {
      const response = await fetch(`${config.apiUrl}/api/users/me/nwc-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ nwc_connection_uri: uri })
      })

      if (!response.ok) {
        const data = await response.json().catch(() => ({}))
        console.error('[OAuthCallback] Backend rejected NWC connection:', response.status, data)
        setError(`Failed to store connection: ${data.message || response.statusText}`)
        return
      }

      console.log('[OAuthCallback] NWC connection stored successfully')
    } catch (err) {
      console.error('[OAuthCallback] Failed to store NWC connection:', err)
      setError(`Network error storing connection: ${err.message}`)
      return
    }

    const returnTo = sessionStorage.getItem('oauth_return_to') || '/'
    sessionStorage.removeItem('oauth_return_to')
    navigate(returnTo, { replace: true })
  }, [token, navigate])

  useEffect(() => {
    if (nwcConnectionUri) {
      storeAndRedirect(nwcConnectionUri)
    }
  }, [nwcConnectionUri, storeAndRedirect])

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center max-w-md">
          <div className="text-red-500 text-4xl mb-4">!</div>
          <h2 className="text-xl font-bold text-gray-900 mb-2">Wallet Connection Failed</h2>
          <p className="text-red-600 text-sm mb-4">{error}</p>
          <p className="text-gray-500 text-xs mb-4">Check browser console for details.</p>
          <button
            onClick={() => {
              const returnTo = sessionStorage.getItem('oauth_return_to') || '/'
              sessionStorage.removeItem('oauth_return_to')
              navigate(returnTo, { replace: true })
            }}
            className="btn-primary"
          >
            Go Back
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-uma-600 mx-auto mb-4"></div>
        <p className="text-gray-600">{status}</p>
      </div>
    </div>
  )
}

export default OAuthCallback
