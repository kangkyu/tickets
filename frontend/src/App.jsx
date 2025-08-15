import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import EventList from './components/EventList'
import EventDetails from './components/EventDetails'
import TicketPurchase from './components/TicketPurchase'
import PaymentStatus from './components/PaymentStatus'
import TicketList from './components/TicketList'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<EventList />} />
        <Route path="/events/:eventId" element={<EventDetails />} />
        <Route path="/events/:eventId/purchase" element={<TicketPurchase />} />
        <Route path="/tickets/:ticketId/payment" element={<PaymentStatus />} />
        <Route path="/tickets" element={<TicketList />} />
      </Routes>
    </Layout>
  )
}

export default App

