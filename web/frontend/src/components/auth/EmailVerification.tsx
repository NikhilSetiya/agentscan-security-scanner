import React, { useState, useEffect } from 'react'
import { useAuthContext } from '../../contexts/AuthContext'
import { useOnboarding } from '../../hooks/useUserManagement'
import { Button } from '../ui/button'
import { Alert, AlertDescription } from '../ui/alert'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Mail, CheckCircle, AlertCircle, RefreshCw, Loader2 } from 'lucide-react'

interface EmailVerificationProps {
  onVerified?: () => void
  showDismiss?: boolean
  onDismiss?: () => void
}

export function EmailVerification({ onVerified, showDismiss, onDismiss }: EmailVerificationProps) {
  const { user, isEmailConfirmed } = useAuthContext()
  const { sendEmailVerification, isLoading, error } = useOnboarding()
  const [emailSent, setEmailSent] = useState(false)
  const [countdown, setCountdown] = useState(0)

  // Auto-check verification status
  useEffect(() => {
    if (isEmailConfirmed && onVerified) {
      onVerified()
    }
  }, [isEmailConfirmed, onVerified])

  // Countdown timer for resend button
  useEffect(() => {
    let timer: NodeJS.Timeout
    if (countdown > 0) {
      timer = setTimeout(() => setCountdown(countdown - 1), 1000)
    }
    return () => clearTimeout(timer)
  }, [countdown])

  const handleSendVerification = async () => {
    const result = await sendEmailVerification()
    if (result.success) {
      setEmailSent(true)
      setCountdown(60) // 60 second cooldown
    }
  }

  if (isEmailConfirmed) {
    return (
      <Alert className="border-green-200 bg-green-50">
        <CheckCircle className="h-4 w-4 text-green-600" />
        <AlertDescription className="text-green-800">
          Your email address has been verified successfully!
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <Card className="border-yellow-200 bg-yellow-50">
      <CardHeader>
        <CardTitle className="flex items-center text-yellow-800">
          <Mail className="h-5 w-5 mr-2" />
          Email Verification Required
        </CardTitle>
        <CardDescription className="text-yellow-700">
          Please verify your email address to access all features of AgentScan.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <Alert className="border-red-200 bg-red-50">
            <AlertCircle className="h-4 w-4 text-red-600" />
            <AlertDescription className="text-red-800">
              {error}
            </AlertDescription>
          </Alert>
        )}

        {emailSent && (
          <Alert className="border-blue-200 bg-blue-50">
            <Mail className="h-4 w-4 text-blue-600" />
            <AlertDescription className="text-blue-800">
              Verification email sent to <strong>{user?.email}</strong>. 
              Please check your inbox and spam folder.
            </AlertDescription>
          </Alert>
        )}

        <div className="space-y-3">
          <p className="text-sm text-yellow-700">
            We've sent a verification link to <strong>{user?.email}</strong>. 
            Click the link in the email to verify your account.
          </p>

          <div className="flex flex-col sm:flex-row gap-3">
            <Button
              onClick={handleSendVerification}
              disabled={isLoading || countdown > 0}
              variant="outline"
              size="sm"
            >
              {isLoading ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Sending...
                </>
              ) : countdown > 0 ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Resend in {countdown}s
                </>
              ) : (
                <>
                  <RefreshCw className="h-4 w-4 mr-2" />
                  {emailSent ? 'Resend Email' : 'Send Verification Email'}
                </>
              )}
            </Button>

            {showDismiss && onDismiss && (
              <Button
                onClick={onDismiss}
                variant="ghost"
                size="sm"
              >
                Dismiss
              </Button>
            )}
          </div>

          <div className="text-xs text-yellow-600 space-y-1">
            <p>• Check your spam or junk folder if you don't see the email</p>
            <p>• Make sure {user?.email} is correct</p>
            <p>• The verification link expires in 24 hours</p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// Compact version for banners/notifications
export function EmailVerificationBanner({ onDismiss }: { onDismiss?: () => void }) {
  const { user, isEmailConfirmed } = useAuthContext()
  const { sendEmailVerification, isLoading } = useOnboarding()
  const [countdown, setCountdown] = useState(0)

  useEffect(() => {
    let timer: NodeJS.Timeout
    if (countdown > 0) {
      timer = setTimeout(() => setCountdown(countdown - 1), 1000)
    }
    return () => clearTimeout(timer)
  }, [countdown])

  const handleSendVerification = async () => {
    const result = await sendEmailVerification()
    if (result.success) {
      setCountdown(60)
    }
  }

  if (isEmailConfirmed) {
    return null
  }

  return (
    <div className="bg-yellow-50 border-l-4 border-yellow-400 p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center">
          <AlertCircle className="h-5 w-5 text-yellow-400 mr-3" />
          <div>
            <p className="text-sm font-medium text-yellow-800">
              Email verification required
            </p>
            <p className="text-sm text-yellow-700">
              Please verify {user?.email} to access all features.
            </p>
          </div>
        </div>
        
        <div className="flex items-center space-x-3">
          <Button
            onClick={handleSendVerification}
            disabled={isLoading || countdown > 0}
            variant="outline"
            size="sm"
          >
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : countdown > 0 ? (
              `Resend (${countdown}s)`
            ) : (
              'Send Email'
            )}
          </Button>
          
          {onDismiss && (
            <Button
              onClick={onDismiss}
              variant="ghost"
              size="sm"
            >
              ×
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}