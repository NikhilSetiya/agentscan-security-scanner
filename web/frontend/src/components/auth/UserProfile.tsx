import React, { useState, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useAuthContext } from '../../contexts/AuthContext'
import { useUserProfile, useUserActivity, useAccountManagement } from '../../hooks/useUserManagement'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Alert, AlertDescription } from '../ui/alert'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../ui/tabs'
import { Badge } from '../ui/badge'
import { Loader2, User, Mail, Shield, Key, CheckCircle, AlertCircle, Camera, Upload, Activity, Settings, Trash2 } from 'lucide-react'

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
  const { profile, isLoading: profileLoading, updateProfile, updatePreferences, uploadAvatar } = useUserProfile()
  const { activity, isLoading: activityLoading } = useUserActivity()
  const { deactivateAccount, deleteAccount, isLoading: accountLoading } = useAccountManagement()
  
  const [isUpdatingPassword, setIsUpdatingPassword] = useState(false)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [passwordSuccess, setPasswordSuccess] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  
  const fileInputRef = useRef<HTMLInputElement>(null)

  const profileForm = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      fullName: profile?.full_name || user?.name || '',
      email: profile?.email || user?.email || ''
    }
  })

  // Update form when profile loads
  React.useEffect(() => {
    if (profile) {
      profileForm.reset({
        fullName: profile.full_name || '',
        email: profile.email || ''
      })
    }
  }, [profile, profileForm])

  const passwordForm = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema)
  })

  const onProfileSubmit = async (data: ProfileFormData) => {
    const result = await updateProfile({
      full_name: data.fullName,
      email: data.email
    })

    if (result.success) {
      // Also update Supabase auth user
      await updateUser({
        email: data.email,
        data: {
          full_name: data.fullName
        }
      })
    }
  }

  const handleAvatarUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    // Validate file type and size
    if (!file.type.startsWith('image/')) {
      alert('Please select an image file')
      return
    }

    if (file.size > 5 * 1024 * 1024) { // 5MB limit
      alert('File size must be less than 5MB')
      return
    }

    await uploadAvatar(file)
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

  if (!user || profileLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }

  const currentProfile = profile || {
    id: user.id,
    email: user.email,
    full_name: user.name,
    avatar_url: null,
    role: user.role,
    organization_id: null,
    preferences: {
      theme: 'system' as const,
      notifications: {
        email: true,
        push: true,
        scan_completion: true,
        security_alerts: true,
        weekly_reports: false
      },
      dashboard: {
        default_view: 'overview' as const,
        items_per_page: 25,
        auto_refresh: true
      },
      security: {
        two_factor_enabled: false,
        session_timeout: 60,
        require_password_change: false
      }
    },
    created_at: user.createdAt,
    updated_at: user.updatedAt,
    last_sign_in_at: user.lastSignInAt,
    email_confirmed_at: user.emailConfirmed ? new Date().toISOString() : null,
    is_active: true
  }

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-6">
      <div className="flex items-center space-x-4">
        <div className="relative">
          <div className="h-20 w-20 bg-gray-200 rounded-full flex items-center justify-center overflow-hidden">
            {currentProfile.avatar_url ? (
              <img 
                src={currentProfile.avatar_url} 
                alt="Profile" 
                className="h-full w-full object-cover"
              />
            ) : (
              <User className="h-10 w-10 text-gray-600" />
            )}
          </div>
          <button
            onClick={() => fileInputRef.current?.click()}
            className="absolute -bottom-1 -right-1 h-8 w-8 bg-blue-600 text-white rounded-full flex items-center justify-center hover:bg-blue-700 transition-colors"
          >
            <Camera className="h-4 w-4" />
          </button>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            onChange={handleAvatarUpload}
            className="hidden"
          />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{currentProfile.full_name || 'User Profile'}</h1>
          <p className="text-gray-600">{currentProfile.email}</p>
          <div className="flex items-center space-x-2 mt-2">
            <Badge className={getRoleColor(currentProfile.role)}>
              {currentProfile.role}
            </Badge>
            {currentProfile.email_confirmed_at ? (
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
            {currentProfile.is_active && (
              <Badge className="bg-green-100 text-green-800">
                Active
              </Badge>
            )}
          </div>
        </div>
      </div>

      <Tabs defaultValue="profile" className="space-y-6">
        <TabsList>
          <TabsTrigger value="profile">Profile</TabsTrigger>
          <TabsTrigger value="preferences">Preferences</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
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

        <TabsContent value="preferences">
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center">
                  <Settings className="h-5 w-5 mr-2" />
                  Appearance
                </CardTitle>
                <CardDescription>
                  Customize how the application looks and feels.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <Label>Theme</Label>
                  <select 
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                    value={currentProfile.preferences.theme}
                    onChange={(e) => updatePreferences({ 
                      theme: e.target.value as 'light' | 'dark' | 'system' 
                    })}
                  >
                    <option value="light">Light</option>
                    <option value="dark">Dark</option>
                    <option value="system">System</option>
                  </select>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Notifications</CardTitle>
                <CardDescription>
                  Choose what notifications you want to receive.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Email Notifications</Label>
                    <p className="text-sm text-gray-500">Receive notifications via email</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={currentProfile.preferences.notifications.email}
                    onChange={(e) => updatePreferences({
                      notifications: {
                        ...currentProfile.preferences.notifications,
                        email: e.target.checked
                      }
                    })}
                    className="h-4 w-4 text-blue-600 border-gray-300 rounded"
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <Label>Scan Completion</Label>
                    <p className="text-sm text-gray-500">Get notified when scans complete</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={currentProfile.preferences.notifications.scan_completion}
                    onChange={(e) => updatePreferences({
                      notifications: {
                        ...currentProfile.preferences.notifications,
                        scan_completion: e.target.checked
                      }
                    })}
                    className="h-4 w-4 text-blue-600 border-gray-300 rounded"
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <Label>Security Alerts</Label>
                    <p className="text-sm text-gray-500">Important security notifications</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={currentProfile.preferences.notifications.security_alerts}
                    onChange={(e) => updatePreferences({
                      notifications: {
                        ...currentProfile.preferences.notifications,
                        security_alerts: e.target.checked
                      }
                    })}
                    className="h-4 w-4 text-blue-600 border-gray-300 rounded"
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <Label>Weekly Reports</Label>
                    <p className="text-sm text-gray-500">Weekly summary of your scans</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={currentProfile.preferences.notifications.weekly_reports}
                    onChange={(e) => updatePreferences({
                      notifications: {
                        ...currentProfile.preferences.notifications,
                        weekly_reports: e.target.checked
                      }
                    })}
                    className="h-4 w-4 text-blue-600 border-gray-300 rounded"
                  />
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Dashboard</CardTitle>
                <CardDescription>
                  Customize your dashboard experience.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <Label>Default View</Label>
                  <select 
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                    value={currentProfile.preferences.dashboard.default_view}
                    onChange={(e) => updatePreferences({
                      dashboard: {
                        ...currentProfile.preferences.dashboard,
                        default_view: e.target.value as 'overview' | 'scans' | 'findings'
                      }
                    })}
                  >
                    <option value="overview">Overview</option>
                    <option value="scans">Scans</option>
                    <option value="findings">Findings</option>
                  </select>
                </div>

                <div>
                  <Label>Items per Page</Label>
                  <select 
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                    value={currentProfile.preferences.dashboard.items_per_page}
                    onChange={(e) => updatePreferences({
                      dashboard: {
                        ...currentProfile.preferences.dashboard,
                        items_per_page: parseInt(e.target.value)
                      }
                    })}
                  >
                    <option value="10">10</option>
                    <option value="25">25</option>
                    <option value="50">50</option>
                    <option value="100">100</option>
                  </select>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <Label>Auto Refresh</Label>
                    <p className="text-sm text-gray-500">Automatically refresh data</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={currentProfile.preferences.dashboard.auto_refresh}
                    onChange={(e) => updatePreferences({
                      dashboard: {
                        ...currentProfile.preferences.dashboard,
                        auto_refresh: e.target.checked
                      }
                    })}
                    className="h-4 w-4 text-blue-600 border-gray-300 rounded"
                  />
                </div>
              </CardContent>
            </Card>
          </div>
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

        <TabsContent value="activity">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Activity className="h-5 w-5 mr-2" />
                Recent Activity
              </CardTitle>
              <CardDescription>
                Your recent account activity and login history.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {activityLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin" />
                </div>
              ) : activity.length > 0 ? (
                <div className="space-y-4">
                  {activity.slice(0, 10).map((item, index) => (
                    <div key={index} className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
                      <div>
                        <p className="font-medium text-sm">{item.action.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}</p>
                        <p className="text-xs text-gray-500">
                          {new Date(item.created_at).toLocaleString()}
                        </p>
                        {item.ip_address && (
                          <p className="text-xs text-gray-400">IP: {item.ip_address}</p>
                        )}
                      </div>
                      {item.details && Object.keys(item.details).length > 0 && (
                        <Badge variant="outline" className="text-xs">
                          Details
                        </Badge>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-gray-500">
                  <Activity className="h-12 w-12 mx-auto mb-4 text-gray-300" />
                  <p>No recent activity</p>
                </div>
              )}
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
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label>Account ID</Label>
                  <p className="text-sm text-gray-600 font-mono">{currentProfile.id}</p>
                </div>
                <div>
                  <Label>Provider</Label>
                  <p className="text-sm text-gray-600 capitalize">{user?.provider || 'supabase'}</p>
                </div>
                <div>
                  <Label>Created</Label>
                  <p className="text-sm text-gray-600">
                    {new Date(currentProfile.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div>
                  <Label>Last Updated</Label>
                  <p className="text-sm text-gray-600">
                    {new Date(currentProfile.updated_at).toLocaleDateString()}
                  </p>
                </div>
                {currentProfile.last_sign_in_at && (
                  <div>
                    <Label>Last Sign In</Label>
                    <p className="text-sm text-gray-600">
                      {new Date(currentProfile.last_sign_in_at).toLocaleString()}
                    </p>
                  </div>
                )}
                <div>
                  <Label>Account Status</Label>
                  <p className="text-sm text-gray-600">
                    {currentProfile.is_active ? 'Active' : 'Inactive'}
                  </p>
                </div>
              </div>

              <div className="pt-6 border-t space-y-4">
                <div className="flex flex-col sm:flex-row gap-3">
                  <Button
                    onClick={signOut}
                    variant="outline"
                    className="w-full sm:w-auto"
                  >
                    Sign Out
                  </Button>
                  
                  <Button
                    onClick={() => deactivateAccount()}
                    variant="outline"
                    disabled={accountLoading}
                    className="w-full sm:w-auto"
                  >
                    {accountLoading ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Processing...
                      </>
                    ) : (
                      'Deactivate Account'
                    )}
                  </Button>
                </div>

                <div className="pt-4 border-t border-red-200">
                  <h4 className="text-sm font-medium text-red-900 mb-2">Danger Zone</h4>
                  <p className="text-sm text-red-600 mb-4">
                    Once you delete your account, there is no going back. Please be certain.
                  </p>
                  
                  {!showDeleteConfirm ? (
                    <Button
                      onClick={() => setShowDeleteConfirm(true)}
                      variant="destructive"
                      className="w-full sm:w-auto"
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete Account
                    </Button>
                  ) : (
                    <div className="space-y-3">
                      <Alert className="border-red-200 bg-red-50">
                        <AlertCircle className="h-4 w-4" />
                        <AlertDescription className="text-red-800">
                          This action cannot be undone. This will permanently delete your account and remove all associated data.
                        </AlertDescription>
                      </Alert>
                      
                      <div className="flex gap-3">
                        <Button
                          onClick={() => setShowDeleteConfirm(false)}
                          variant="outline"
                          size="sm"
                        >
                          Cancel
                        </Button>
                        <Button
                          onClick={() => deleteAccount()}
                          variant="destructive"
                          size="sm"
                          disabled={accountLoading}
                        >
                          {accountLoading ? (
                            <>
                              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                              Deleting...
                            </>
                          ) : (
                            'Yes, Delete My Account'
                          )}
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}