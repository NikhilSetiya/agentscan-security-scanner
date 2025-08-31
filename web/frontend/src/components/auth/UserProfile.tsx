import React, { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useAuthContext } from '../../contexts/AuthContext'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Alert, AlertDescription } from '../ui/alert'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../ui/tabs'
import { Badge } from '../ui/badge'
import { Loader2, User, Mail, Shield, Key, CheckCircle, AlertCircle } from 'lucide-react'

const profileSchema = z.object({
  fullName: z.string().min(2, 'Full name must be at least 2 characters'),
  email: z.string().email('Please enter a valid email address')
})

const passwordSchema = z.object({
  currentPassword: z.string().min(1, 'Current password is required'),
  newPassword: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .regex(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/, 'Password must contain at least one uppercase letter, one lowercase letter, and one number'),
  confirmPassword: z.string()
}).refine((data) => data.newPassword === data.confirmPassword, {
  message: "Passwords don't match",
  path: ["confirmPassword"]
})

type ProfileFormData = z.infer<typeof profileSchema>
type PasswordFormData = z.infer<typeof passwordSchema>

export function UserProfile() {
  const { user, updateUser, signOut } = useAuthContext()
  const [isUpdatingProfile, setIsUpdatingProfile] = useState(false)
  const [isUpdatingPassword, setIsUpdatingPassword] = useState(false)
  const [profileError, setProfileError] = useState<string | null>(null)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [profileSuccess, setProfileSuccess] = useState(false)
  const [passwordSuccess, setPasswordSuccess] = useState(false)

  const profileForm = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      fullName: user?.name || '',
      email: user?.email || ''
    }
  })

  const passwordForm = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema)
  })

  const onProfileSubmit = async (data: ProfileFormData) => {
    setIsUpdatingProfile(true)
    setProfileError(null)
    setProfileSuccess(false)

    try {
      const result = await updateUser({
        email: data.email,
        data: {
          full_name: data.fullName
        }
      })
      
      if (result.error) {
        setProfileError(result.error.message || 'Failed to update profile')
      } else {
        setProfileSuccess(true)
        setTimeout(() => setProfileSuccess(false), 3000)
      }
    } catch (err) {
      setProfileError('An unexpected error occurred')
    } finally {
      setIsUpdatingProfile(false)
    }
  }

  const onPasswordSubmit = async (data: PasswordFormData) => {
    setIsUpdatingPassword(true)
    setPasswordError(null)
    setPasswordSuccess(false)

    try {
      const result = await updateUser({
        password: data.newPassword
      })
      
      if (result.error) {
        setPasswordError(result.error.message || 'Failed to update password')
      } else {
        setPasswordSuccess(true)
        passwordForm.reset()
        setTimeout(() => setPasswordSuccess(false), 3000)
      }
    } catch (err) {
      setPasswordError('An unexpected error occurred')
    } finally {
      setIsUpdatingPassword(false)
    }
  }

  const getRoleColor = (role: string) => {
    switch (role.toLowerCase()) {
      case 'admin':
      case 'superuser':
        return 'bg-red-100 text-red-800'
      case 'moderator':
        return 'bg-yellow-100 text-yellow-800'
      default:
        return 'bg-blue-100 text-blue-800'
    }
  }

  if (!user) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-6">
      <div className="flex items-center space-x-4">
        <div className="h-16 w-16 bg-gray-200 rounded-full flex items-center justify-center">
          <User className="h-8 w-8 text-gray-600" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{user.name || 'User Profile'}</h1>
          <p className="text-gray-600">{user.email}</p>
          <div className="flex items-center space-x-2 mt-2">
            <Badge className={getRoleColor(user.role)}>
              {user.role}
            </Badge>
            {user.emailConfirmed ? (
              <Badge className="bg-green-100 text-green-800">
                <CheckCircle className="h-3 w-3 mr-1" />
                Verified
              </Badge>
            ) : (
              <Badge className="bg-yellow-100 text-yellow-800">
                <AlertCircle className="h-3 w-3 mr-1" />
                Unverified
              </Badge>
            )}
          </div>
        </div>
      </div>

      <Tabs defaultValue="profile" className="space-y-6">
        <TabsList>
          <TabsTrigger value="profile">Profile</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
          <TabsTrigger value="account">Account</TabsTrigger>
        </TabsList>

        <TabsContent value="profile">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <User className="h-5 w-5 mr-2" />
                Profile Information
              </CardTitle>
              <CardDescription>
                Update your personal information and preferences.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {profileError && (
                <Alert className="mb-6 border-red-200 bg-red-50">
                  <AlertDescription className="text-red-800">
                    {profileError}
                  </AlertDescription>
                </Alert>
              )}

              {profileSuccess && (
                <Alert className="mb-6 border-green-200 bg-green-50">
                  <AlertDescription className="text-green-800">
                    Profile updated successfully!
                  </AlertDescription>
                </Alert>
              )}

              <form onSubmit={profileForm.handleSubmit(onProfileSubmit)} className="space-y-4">
                <div>
                  <Label htmlFor="fullName">Full Name</Label>
                  <Input
                    id="fullName"
                    {...profileForm.register('fullName')}
                    className={profileForm.formState.errors.fullName ? 'border-red-300' : ''}
                  />
                  {profileForm.formState.errors.fullName && (
                    <p className="mt-1 text-sm text-red-600">
                      {profileForm.formState.errors.fullName.message}
                    </p>
                  )}
                </div>

                <div>
                  <Label htmlFor="email">Email Address</Label>
                  <Input
                    id="email"
                    type="email"
                    {...profileForm.register('email')}
                    className={profileForm.formState.errors.email ? 'border-red-300' : ''}
                  />
                  {profileForm.formState.errors.email && (
                    <p className="mt-1 text-sm text-red-600">
                      {profileForm.formState.errors.email.message}
                    </p>
                  )}
                  <p className="mt-1 text-sm text-gray-500">
                    Changing your email will require verification.
                  </p>
                </div>

                <Button
                  type="submit"
                  disabled={isUpdatingProfile}
                  className="w-full sm:w-auto"
                >
                  {isUpdatingProfile ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Updating...
                    </>
                  ) : (
                    'Update Profile'
                  )}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Key className="h-5 w-5 mr-2" />
                Change Password
              </CardTitle>
              <CardDescription>
                Update your password to keep your account secure.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {passwordError && (
                <Alert className="mb-6 border-red-200 bg-red-50">
                  <AlertDescription className="text-red-800">
                    {passwordError}
                  </AlertDescription>
                </Alert>
              )}

              {passwordSuccess && (
                <Alert className="mb-6 border-green-200 bg-green-50">
                  <AlertDescription className="text-green-800">
                    Password updated successfully!
                  </AlertDescription>
                </Alert>
              )}

              <form onSubmit={passwordForm.handleSubmit(onPasswordSubmit)} className="space-y-4">
                <div>
                  <Label htmlFor="currentPassword">Current Password</Label>
                  <Input
                    id="currentPassword"
                    type="password"
                    {...passwordForm.register('currentPassword')}
                    className={passwordForm.formState.errors.currentPassword ? 'border-red-300' : ''}
                  />
                  {passwordForm.formState.errors.currentPassword && (
                    <p className="mt-1 text-sm text-red-600">
                      {passwordForm.formState.errors.currentPassword.message}
                    </p>
                  )}
                </div>

                <div>
                  <Label htmlFor="newPassword">New Password</Label>
                  <Input
                    id="newPassword"
                    type="password"
                    {...passwordForm.register('newPassword')}
                    className={passwordForm.formState.errors.newPassword ? 'border-red-300' : ''}
                  />
                  {passwordForm.formState.errors.newPassword && (
                    <p className="mt-1 text-sm text-red-600">
                      {passwordForm.formState.errors.newPassword.message}
                    </p>
                  )}
                </div>

                <div>
                  <Label htmlFor="confirmPassword">Confirm New Password</Label>
                  <Input
                    id="confirmPassword"
                    type="password"
                    {...passwordForm.register('confirmPassword')}
                    className={passwordForm.formState.errors.confirmPassword ? 'border-red-300' : ''}
                  />
                  {passwordForm.formState.errors.confirmPassword && (
                    <p className="mt-1 text-sm text-red-600">
                      {passwordForm.formState.errors.confirmPassword.message}
                    </p>
                  )}
                </div>

                <Button
                  type="submit"
                  disabled={isUpdatingPassword}
                  className="w-full sm:w-auto"
                >
                  {isUpdatingPassword ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Updating...
                    </>
                  ) : (
                    'Update Password'
                  )}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="account">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Shield className="h-5 w-5 mr-2" />
                Account Information
              </CardTitle>
              <CardDescription>
                View your account details and manage your account.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label>Account ID</Label>
                  <p className="text-sm text-gray-600 font-mono">{user.id}</p>
                </div>
                <div>
                  <Label>Provider</Label>
                  <p className="text-sm text-gray-600 capitalize">{user.provider}</p>
                </div>
                <div>
                  <Label>Created</Label>
                  <p className="text-sm text-gray-600">
                    {new Date(user.createdAt).toLocaleDateString()}
                  </p>
                </div>
                <div>
                  <Label>Last Updated</Label>
                  <p className="text-sm text-gray-600">
                    {new Date(user.updatedAt).toLocaleDateString()}
                  </p>
                </div>
                {user.lastSignInAt && (
                  <div>
                    <Label>Last Sign In</Label>
                    <p className="text-sm text-gray-600">
                      {new Date(user.lastSignInAt).toLocaleString()}
                    </p>
                  </div>
                )}
              </div>

              <div className="pt-6 border-t">
                <Button
                  onClick={signOut}
                  variant="outline"
                  className="w-full sm:w-auto"
                >
                  Sign Out
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}