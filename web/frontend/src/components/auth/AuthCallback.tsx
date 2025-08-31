import React, { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { supabase } from '../../lib/supabase'
import { Loader2, AlertCircle } from 'lucide-react'
import { Alert, AlertDescription } from '../ui/alert'
import { Button } from '../ui/button'

export function AuthCallback() {
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()

  useEffect(() => {
    const handleAuthCallback = async () => {
      try {
        // Handle the OAuth callback
        const { data, error } = await supabase.auth.getSession()
        
        if (error) {
          console.error('Auth callback error:', error)
          setError(error.message)
          return
        }

        if (data.session) {
          // Successfully authenticated
          console.log('Authentication successful:', data.session.user.email)
          
          // Check if there's a redirect URL in the state
          const redirectTo = searchParams.get('redirect_to') || '/dashboard'
          
          // Small delay to ensure auth state is properly set
          setTimeout(() => {
            navigate(redirectTo, { replace: true })
          }, 1000)
        } else {
          // No session found
          setError('No authentication session found')
        }
      } catch (err) {
        console.error('Unexpected error in auth callback:', err)
        setError('An unexpected error occurred during authentication')
      } finally {
        setIsLoading(false)
      }
    }

    handleAuthCallback()
  }, [navigate, searchParams])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full mx-auto text-center">
          <Loader2 className="mx-auto h-12 w-12 animate-spin text-blue-600" />
          <h2 className="mt-4 text-xl font-semibold text-gray-900">
            Completing authentication...
          </h2>
          <p className="mt-2 text-gray-600">
            Please wait while we sign you in.
          </p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full mx-auto">
          <div className="text-center mb-6">
            <AlertCircle className="mx-auto h-12 w-12 text-red-500" />
            <h2 className="mt-4 text-xl font-semibold text-gray-900">
              Authentication Failed
            </h2>
          </div>

          <Alert className="mb-6 border-red-200 bg-red-50">
            <AlertDescription className="text-red-800">
              {error}
            </AlertDescription>
          </Alert>

          <div className="space-y-3">
            <Button
              onClick={() => navigate('/login')}
              className="w-full"
            >
              Try Again
            </Button>
            
            <Button
              onClick={() => navigate('/')}
              variant="outline"
              className="w-full"
            >
              Go Home
            </Button>
          </div>
        </div>
      </div>
    )
  }

  return null
}