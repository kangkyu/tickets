import { useState } from 'react'
import { X, Download, Mail } from 'lucide-react'

const QRCodeDisplay = ({ ticket, onClose }) => {
  const [isDownloading, setIsDownloading] = useState(false)
  const [isEmailing, setIsEmailing] = useState(false)

  // Generate QR code data - just the ticket code for scanners
  const qrData = ticket.ticket_code

  // For now, we'll use a placeholder QR code
  // In production, you'd want to use a proper QR code library like qrcode.react
  const generateQRCode = () => {
    // This generates a QR code with just the ticket code for easy scanning
    return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(qrData)}`
  }

  const handleDownload = async () => {
    setIsDownloading(true)
    try {
      const response = await fetch(generateQRCode())
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `ticket-${ticket.ticket_code}.png`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
    } catch (error) {
      console.error('Failed to download QR code:', error)
    } finally {
      setIsDownloading(false)
    }
  }

  const handleEmail = async () => {
    setIsEmailing(true)
    try {
      // In production, this would send the QR code via email
      // For now, we'll just open the user's email client
      const subject = encodeURIComponent(`Your Ticket for ${ticket.event?.title}`)
      const body = encodeURIComponent(`
Hi there!

Here's your ticket for ${ticket.event?.title}:

Ticket ID: ${ticket.id}
Ticket Code: ${ticket.ticket_code}
Event: ${ticket.event?.title}
Date: ${ticket.event?.start_time ? new Date(ticket.event.start_time).toLocaleDateString() : 'TBD'}

Please present this ticket at the event entrance.

Best regards,
UMA Tickets Team
      `)
      
      window.open(`mailto:?subject=${subject}&body=${body}`)
    } catch (error) {
      console.error('Failed to open email client:', error)
    } finally {
      setIsEmailing(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg max-w-md w-full p-6">
        {/* Header */}
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold text-gray-900">Ticket QR Code</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Event Info */}
        <div className="mb-4 p-3 bg-gray-50 rounded-lg">
          <h3 className="font-semibold text-gray-900">{ticket.event?.title}</h3>
          <p className="text-sm text-gray-600">
            {ticket.event?.start_time ? new Date(ticket.event.start_time).toLocaleDateString() : 'Date TBD'}
          </p>
        </div>

        {/* QR Code */}
        <div className="text-center mb-6">
          <div className="mb-2">
            <p className="text-sm text-gray-600">
              This QR code contains just the ticket code for easy scanning
            </p>
          </div>
          <div className="inline-block p-4 bg-white border-2 border-gray-200 rounded-lg">
            <img
              src={generateQRCode()}
              alt="Ticket QR Code"
              className="w-48 h-48"
            />
          </div>
          <p className="text-xs text-gray-500 mt-2">
            QR code contains ticket code: {ticket.ticket_code}
          </p>
        </div>

        {/* Ticket Details */}
        <div className="mb-6 space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-gray-500">Ticket ID:</span>
            <span className="font-mono text-gray-900">{ticket.id}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-500">Ticket Code:</span>
            <span className="font-mono text-gray-900">{ticket.ticket_code}</span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3">
          <button
            onClick={handleDownload}
            disabled={isDownloading}
            className="flex-1 btn-secondary flex items-center justify-center gap-2"
          >
            <Download className="w-4 h-4" />
            {isDownloading ? 'Downloading...' : 'Download'}
          </button>
          <button
            onClick={handleEmail}
            disabled={isEmailing}
            className="flex-1 btn-secondary flex items-center justify-center gap-2"
          >
            <Mail className="w-4 h-4" />
            {isEmailing ? 'Opening...' : 'Email'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default QRCodeDisplay
