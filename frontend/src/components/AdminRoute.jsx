import { useAuth } from '../contexts/AuthContext'
import { Navigate } from 'react-router-dom'

const AdminRoute = ({ children }) => {
  const { user, isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-500"></div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  // Check if user is admin (based on backend admin middleware logic)
  const adminEmails = ['admin@example.com'] // This matches the default from backend config
  const isAdmin = user && adminEmails.includes(user.email)

  if (!isAdmin) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="max-w-md mx-auto text-center">
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
            <h2 className="text-xl font-bold mb-2">Access Denied</h2>
            <p>You don't have admin privileges to access this page.</p>
          </div>
        </div>
      </div>
    )
  }

  return children
}

export default AdminRoute