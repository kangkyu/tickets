import { useState, useEffect } from 'react'
import { Copy, Check, Download } from 'lucide-react'
import QRCode from 'qrcode'

const QRCodeDisplay = ({ bolt11Invoice, amount, onCopy }) => {
  const [qrCodeDataUrl, setQrCodeDataUrl] = useState('')
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')

  // Generate QR code when invoice changes
  useEffect(() => {
    if (bolt11Invoice) {
      generateQRCode(bolt11Invoice)
    }
  }, [bolt11Invoice])

  const generateQRCode = async (invoice) => {
    try {
      setError('')
      const dataUrl = await QRCode.toDataURL(invoice, {
        width: 256,
        margin: 2,
        color: {
          dark: '#000000',
          light: '#FFFFFF'
        }
      })
      setQrCodeDataUrl(dataUrl)
    } catch (err) {
      setError('Failed to generate QR code')
      console.error('QR code generation error:', err)
    }
  }

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(bolt11Invoice)
      setCopied(true)
      
      // Call parent callback if provided
      if (onCopy) {
        onCopy(bolt11Invoice)
      }
      
      // Reset copied state after 2 seconds
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy to clipboard:', err)
      // Fallback for older browsers
      const textArea = document.createElement('textarea')
      textArea.value = bolt11Invoice
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
      
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const downloadQRCode = () => {
    if (!qrCodeDataUrl) return
    
    const link = document.createElement('a')
    link.download = `lightning-invoice-${Date.now()}.png`
    link.href = qrCodeDataUrl
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  if (error) {
    return (
      <div className="text-center p-6 bg-red-50 rounded-lg border border-red-200">
        <div className="text-red-500 text-6xl mb-2">⚠️</div>
        <p className="text-red-700 font-medium">{error}</p>
        <button
          onClick={() => generateQRCode(bolt11Invoice)}
          className="btn-primary mt-3"
        >
          Try Again
        </button>
      </div>
    )
  }

  return (
    <div className="text-center space-y-4">
      {/* QR Code */}
      <div className="flex justify-center">
        <div className="relative">
          {qrCodeDataUrl ? (
            <img
              src={qrCodeDataUrl}
              alt="Lightning Invoice QR Code"
              className="w-64 h-64 border-4 border-gray-200 rounded-lg"
            />
          ) : (
            <div className="w-64 h-64 bg-gray-100 border-4 border-gray-200 rounded-lg flex items-center justify-center">
              <div className="text-gray-400">Generating QR code...</div>
            </div>
          )}
          
          {/* Amount Overlay */}
          {amount && (
            <div className="absolute bottom-2 left-1/2 transform -translate-x-1/2 bg-white px-3 py-1 rounded-full shadow-lg border border-gray-200">
              <span className="text-sm font-semibold text-gray-900">
                {amount} sats
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Invoice Text */}
      <div className="space-y-3">
        <p className="text-sm text-gray-600">
          Scan this QR code with your Lightning wallet to pay
        </p>
        
        {/* Invoice Display */}
        <div className="bg-gray-50 p-3 rounded-lg">
          <p className="text-xs text-gray-500 mb-1">Lightning Invoice</p>
          <div className="flex items-center justify-between">
            <code className="text-xs text-gray-700 break-all flex-1 text-left mr-2">
              {bolt11Invoice}
            </code>
            <button
              onClick={copyToClipboard}
              className="flex-shrink-0 p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
              title="Copy invoice"
            >
              {copied ? (
                <Check className="w-4 h-4 text-green-500" />
              ) : (
                <Copy className="w-4 h-4" />
              )}
            </button>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex gap-3 justify-center">
          <button
            onClick={copyToClipboard}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
              copied
                ? 'bg-green-100 text-green-700 border border-green-200'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200 border border-gray-200'
            }`}
          >
            {copied ? (
              <>
                <Check className="w-4 h-4" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="w-4 h-4" />
                Copy Invoice
              </>
            )}
          </button>
          
          <button
            onClick={downloadQRCode}
            disabled={!qrCodeDataUrl}
            className="flex items-center gap-2 px-4 py-2 bg-white text-gray-700 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Download className="w-4 h-4" />
            Download QR
          </button>
        </div>
      </div>

      {/* Instructions */}
      <div className="bg-blue-50 p-4 rounded-lg border border-blue-200">
        <h4 className="font-medium text-blue-900 mb-2">How to pay:</h4>
        <ol className="text-sm text-blue-800 space-y-1 text-left">
          <li>1. Open your Lightning wallet app</li>
          <li>2. Scan the QR code above</li>
          <li>3. Confirm the payment amount</li>
          <li>4. Complete the payment</li>
        </ol>
        <p className="text-xs text-blue-600 mt-2">
          Payment will be confirmed automatically once received
        </p>
      </div>
    </div>
  )
}

export default QRCodeDisplay
