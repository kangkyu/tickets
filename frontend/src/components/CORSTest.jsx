import React, { useState } from 'react'
import { apiUtils } from '../services/api'
import config from '../config/api'

const CORSTest = () => {
  const [testResult, setTestResult] = useState(null)
  const [isLoading, setIsLoading] = useState(false)

  const runCORSTest = async () => {
    setIsLoading(true)
    setTestResult(null)
    
    try {
      const result = await apiUtils.testCORS()
      setTestResult(result)
    } catch (error) {
      setTestResult({ success: false, error })
    } finally {
      setIsLoading(false)
    }
  }

  const testWithProxy = async () => {
    setIsLoading(true)
    setTestResult(null)
    
    try {
      // Test with CORS proxy
      const proxyUrl = config.corsProxy + config.apiUrl + '/health'
      console.log('🧪 Testing with CORS proxy:', proxyUrl)
      
      const response = await fetch(proxyUrl, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })
      
      const result = { success: true, response, usedProxy: true }
      setTestResult(result)
    } catch (error) {
      setTestResult({ success: false, error, usedProxy: true })
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="p-6 bg-gray-50 rounded-lg border">
      <h3 className="text-lg font-semibold mb-4">🔍 CORS Connectivity Test</h3>
      
      <div className="space-y-4">
        <div className="text-sm text-gray-600">
          <p><strong>Current Origin:</strong> {window.location.origin}</p>
          <p><strong>Target API:</strong> {config.apiUrl}</p>
          <p><strong>Mode:</strong> {config.isDevelopment ? 'Development' : 'Production'}</p>
        </div>
        
        <div className="flex space-x-3">
          <button
            onClick={runCORSTest}
            disabled={isLoading}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
          >
            {isLoading ? 'Testing...' : 'Test Direct Connection'}
          </button>
          
          {config.isDevelopment && (
            <button
              onClick={testWithProxy}
              disabled={isLoading}
              className="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600 disabled:opacity-50"
            >
              {isLoading ? 'Testing...' : 'Test with CORS Proxy'}
            </button>
          )}
        </div>
        
        {testResult && (
          <div className={`p-4 rounded border ${
            testResult.success ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'
          }`}>
            <h4 className="font-semibold mb-2">
              {testResult.success ? '✅ Test Passed' : '❌ Test Failed'}
              {testResult.usedProxy && ' (with proxy)'}
            </h4>
            
            {testResult.success ? (
              <div className="text-sm">
                <p><strong>Status:</strong> {testResult.response?.status}</p>
                <p><strong>Status Text:</strong> {testResult.response?.statusText}</p>
                <details className="mt-2">
                  <summary className="cursor-pointer font-medium">Response Headers</summary>
                  <pre className="mt-2 text-xs bg-gray-100 p-2 rounded overflow-auto">
                    {JSON.stringify(Object.fromEntries(testResult.response?.headers?.entries() || {}), null, 2)}
                  </pre>
                </details>
              </div>
            ) : (
              <div className="text-sm">
                <p><strong>Error:</strong> {testResult.error?.message}</p>
                {testResult.error?.response && (
                  <details className="mt-2">
                    <summary className="cursor-pointer font-medium">Error Details</summary>
                    <pre className="mt-2 text-xs bg-gray-100 p-2 rounded overflow-auto">
                      {JSON.stringify(testResult.error.response, null, 2)}
                    </pre>
                  </details>
                )}
              </div>
            )}
          </div>
        )}
        
        <div className="text-xs text-gray-500">
          <p><strong>💡 Tips:</strong></p>
          <ul className="list-disc list-inside mt-1 space-y-1">
            <li>Check browser console for detailed CORS error information</li>
            <li>Verify backend CORS middleware is working</li>
            <li>Ensure backend allows your origin: {window.location.origin}</li>
            <li>For development, consider using a CORS proxy</li>
          </ul>
        </div>
      </div>
    </div>
  )
}

export default CORSTest
