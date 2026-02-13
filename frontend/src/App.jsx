import { useEffect, useCallback, useState } from 'react'
import { Routes, Route } from 'react-router-dom'
import { useOAuth } from '@uma-sdk/uma-auth-client'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import Layout from './components/Layout'
import EventList from './components/EventList'
import EventDetails from './components/EventDetails'
import TicketPurchase from './components/TicketPurchase'
import TicketList from './components/TicketList'
import Login from './components/Login'
import ProtectedRoute from './components/ProtectedRoute'
import AdminRoute from './components/AdminRoute'
import AdminDashboard from './components/AdminDashboard'
import CreateEvent from './components/CreateEvent'
import EditEvent from './components/EditEvent'
import config from './config/api'

// Only processes OAuth callback — mounted conditionally when URL has OAuth params
function OAuthCallbackHandler() {
  const { nwcConnectionUri } = useOAuth()
  const { token } = useAuth()

  const storeNWCConnection = useCallback(async (uri) => {
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
  }, [token])

  useEffect(() => {
    if (nwcConnectionUri) {
      storeNWCConnection(nwcConnectionUri)
    }
  }, [nwcConnectionUri, storeNWCConnection])

  return null
}

// Only renders OAuthCallbackHandler when URL contains OAuth callback params
function NWCConnectionHandler({ children }) {
  const [hasOAuthParams] = useState(() => {
    const params = new URLSearchParams(window.location.search)
    return params.has('code') && params.has('state')
  })

  return (
    <>
      {hasOAuthParams && <OAuthCallbackHandler />}
      {children}
    </>
  )
}

function App() {
  return (
    <AuthProvider>
      <NWCConnectionHandler>
      <Layout>

          <Routes>
            {/* Public routes */}
            <Route path="/" element={<EventList />} />
            <Route path="/events/:eventId" element={<EventDetails />} />
            <Route path="/login" element={<Login />} />
            
            {/* Protected routes - require authentication */}
            <Route path="/events/:eventId/purchase" element={
              <ProtectedRoute>
                <TicketPurchase />
              </ProtectedRoute>
            } />
            <Route path="/tickets" element={
              <ProtectedRoute>
                <TicketList />
              </ProtectedRoute>
            } />
            
            {/* Admin routes - require admin privileges */}
            <Route path="/admin" element={
              <AdminRoute>
                <AdminDashboard />
              </AdminRoute>
            } />
            <Route path="/admin/events/new" element={
              <AdminRoute>
                <CreateEvent />
              </AdminRoute>
            } />
            <Route path="/admin/events/:eventId/edit" element={
              <AdminRoute>
                <EditEvent />
              </AdminRoute>
            } />
          </Routes>
      </Layout>
      </NWCConnectionHandler>
    </AuthProvider>
  )
}

export default App

