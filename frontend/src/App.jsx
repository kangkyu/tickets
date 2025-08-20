import { Routes, Route } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import Layout from './components/Layout'
import EventList from './components/EventList'
import EventDetails from './components/EventDetails'
import TicketPurchase from './components/TicketPurchase'
import PaymentStatus from './components/PaymentStatus'
import TicketList from './components/TicketList'
import Login from './components/Login'
import ProtectedRoute from './components/ProtectedRoute'
import AdminRoute from './components/AdminRoute'
import AdminDashboard from './components/AdminDashboard'
import CreateEvent from './components/CreateEvent'
import EditEvent from './components/EditEvent'



function App() {
  return (
    <AuthProvider>
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
            <Route path="/tickets/:ticketId/payment" element={
              <ProtectedRoute>
                <PaymentStatus />
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
    </AuthProvider>
  )
}

export default App

