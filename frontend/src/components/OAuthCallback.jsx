import { useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useOAuth } from '@uma-sdk/uma-auth-client'
import { useAuth } from '../contexts/AuthContext'
import config from '../config/api'

const OAuthCallback = () => {
  const { nwcConnectionUri } = useOAuth()
  const { token } = useAuth()
  const navigate = useNavigate()

  const storeAndRedirect = useCallback(async (uri) => {
    if (!token) return
    try {
      await fetch(`${config.apiUrl}/api/users/me/nwc-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ nwc_connection_uri: uri })
      })
    } catch (error) {
      console.error('Failed to store NWC connection:', error)
    }

    // Redirect back to where the user was
    const returnTo = sessionStorage.getItem('oauth_return_to') || '/'
    sessionStorage.removeItem('oauth_return_to')
    navigate(returnTo, { replace: true })
  }, [token, navigate])

  useEffect(() => {
    if (nwcConnectionUri) {
      storeAndRedirect(nwcConnectionUri)
    }
  }, [nwcConnectionUri, storeAndRedirect])

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-uma-600 mx-auto mb-4"></div>
        <p className="text-gray-600">Connecting wallet...</p>
      </div>
    </div>
  )
}

export default OAuthCallback
