import React, { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useOnboarding } from '../../hooks/useUserManagement'
import { useAuthContext } from '../../contexts/AuthContext'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Alert, AlertDescription } from '../ui/alert'
import { Badge } from '../ui/badge'
import { 
  CheckCircle, 
  ArrowRight, 
  ArrowLeft, 
  User, 
  Building, 
  Target, 
  Users,
  Loader2,
  Sparkles
} from 'lucide-react'

const onboardingSchema = z.object({
  full_name: z.string().min(2, 'Full name must be at least 2 characters'),
  organization_name: z.string().optional(),
  role_in_organization: z.string().optional(),
  use_case: z.string().optional(),
  team_size: z.string().optional()
})

type OnboardingFormData = z.infer<typeof onboardingSchema>

interface OnboardingWizardProps {
  onComplete: () => void
  onSkip?: () => void
}

export function OnboardingWizard({ onComplete, onSkip }: OnboardingWizardProps) {
  const { user } = useAuthContext()
  const { completeOnboarding, isLoading, error } = useOnboarding()
  const [currentStep, setCurrentStep] = useState(0)
  const [completedSteps, setCompletedSteps] = useState<number[]>([])

  const form = useForm<OnboardingFormData>({
    resolver: zodResolver(onboardingSchema),
    defaultValues: {
      full_name: user?.name || '',
      organization_name: '',
      role_in_organization: '',
      use_case: '',
      team_size: ''
    }
  })

  const steps = [
    {
      id: 'personal',
      title: 'Personal Information',
      description: 'Tell us about yourself',
      icon: User,
      fields: ['full_name']
    },
    {
      id: 'organization',
      title: 'Organization',
      description: 'About your organization',
      icon: Building,
      fields: ['organization_name', 'role_in_organization']
    },
    {
      id: 'usage',
      title: 'How will you use AgentScan?',
      description: 'Help us customize your experience',
      icon: Target,
      fields: ['use_case', 'team_size']
    },
    {
      id: 'complete',
      title: 'All Set!',
      description: 'Welcome to AgentScan',
      icon: Sparkles,
      fields: []
    }
  ]

  const currentStepData = steps[currentStep]
  const isLastStep = currentStep === steps.length - 1
  const isFirstStep = currentStep === 0

  const validateCurrentStep = () => {
    const fieldsToValidate = currentStepData.fields
    if (fieldsToValidate.length === 0) return true

    const values = form.getValues()
    return fieldsToValidate.every(field => {
      const value = values[field as keyof OnboardingFormData]
      return value && value.trim().length > 0
    })
  }

  const handleNext = () => {
    if (validateCurrentStep()) {
      setCompletedSteps(prev => [...prev, currentStep])
      if (isLastStep) {
        handleComplete()
      } else {
        setCurrentStep(prev => prev + 1)
      }
    }
  }

  const handlePrevious = () => {
    if (!isFirstStep) {
      setCurrentStep(prev => prev - 1)
    }
  }

  const handleComplete = async () => {
    const formData = form.getValues()
    const result = await completeOnboarding(formData)
    
    if (result.success) {
      onComplete()
    }
  }

  const handleSkip = () => {
    if (onSkip) {
      onSkip()
    } else {
      onComplete()
    }
  }

  const useCaseOptions = [
    'Personal projects',
    'Small team (2-10 people)',
    'Medium team (11-50 people)',
    'Large organization (50+ people)',
    'Enterprise security',
    'Compliance and auditing',
    'Open source projects',
    'Educational purposes'
  ]

  const teamSizeOptions = [
    'Just me',
    '2-5 people',
    '6-10 people',
    '11-25 people',
    '26-50 people',
    '51-100 people',
    '100+ people'
  ]

  const roleOptions = [
    'Developer',
    'Security Engineer',
    'DevOps Engineer',
    'Team Lead',
    'Engineering Manager',
    'CTO/VP Engineering',
    'Security Manager',
    'Compliance Officer',
    'Other'
  ]

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-2xl">
        {/* Progress Bar */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-4">
            {steps.map((step, index) => {
              const Icon = step.icon
              const isCompleted = completedSteps.includes(index)
              const isCurrent = index === currentStep
              
              return (
                <div key={step.id} className="flex items-center">
                  <div className={`
                    flex items-center justify-center w-10 h-10 rounded-full border-2 transition-colors
                    ${isCompleted ? 'bg-green-500 border-green-500 text-white' : 
                      isCurrent ? 'bg-blue-500 border-blue-500 text-white' : 
                      'bg-white border-gray-300 text-gray-400'}
                  `}>
                    {isCompleted ? (
                      <CheckCircle className="w-5 h-5" />
                    ) : (
                      <Icon className="w-5 h-5" />
                    )}
                  </div>
                  {index < steps.length - 1 && (
                    <div className={`
                      w-16 h-0.5 mx-2 transition-colors
                      ${isCompleted ? 'bg-green-500' : 'bg-gray-300'}
                    `} />
                  )}
                </div>
              )
            })}
          </div>
          <div className="text-center">
            <h2 className="text-2xl font-bold text-gray-900">{currentStepData.title}</h2>
            <p className="text-gray-600 mt-1">{currentStepData.description}</p>
          </div>
        </div>

        <Card>
          <CardContent className="p-8">
            {error && (
              <Alert className="mb-6 border-red-200 bg-red-50">
                <AlertDescription className="text-red-800">
                  {error}
                </AlertDescription>
              </Alert>
            )}

            <form className="space-y-6">
              {/* Step 1: Personal Information */}
              {currentStep === 0 && (
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="full_name">Full Name *</Label>
                    <Input
                      id="full_name"
                      {...form.register('full_name')}
                      placeholder="Enter your full name"
                      className={form.formState.errors.full_name ? 'border-red-300' : ''}
                    />
                    {form.formState.errors.full_name && (
                      <p className="mt-1 text-sm text-red-600">
                        {form.formState.errors.full_name.message}
                      </p>
                    )}
                  </div>
                  
                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                    <p className="text-sm text-blue-800">
                      This helps us personalize your experience and communicate with you effectively.
                    </p>
                  </div>
                </div>
              )}

              {/* Step 2: Organization */}
              {currentStep === 1 && (
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="organization_name">Organization Name</Label>
                    <Input
                      id="organization_name"
                      {...form.register('organization_name')}
                      placeholder="Enter your organization name (optional)"
                    />
                    <p className="mt-1 text-sm text-gray-500">
                      Leave blank if you're using AgentScan for personal projects
                    </p>
                  </div>

                  <div>
                    <Label htmlFor="role_in_organization">Your Role</Label>
                    <select
                      id="role_in_organization"
                      {...form.register('role_in_organization')}
                      className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                    >
                      <option value="">Select your role (optional)</option>
                      {roleOptions.map(role => (
                        <option key={role} value={role}>{role}</option>
                      ))}
                    </select>
                  </div>
                </div>
              )}

              {/* Step 3: Usage */}
              {currentStep === 2 && (
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="use_case">Primary Use Case</Label>
                    <select
                      id="use_case"
                      {...form.register('use_case')}
                      className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                    >
                      <option value="">Select your primary use case (optional)</option>
                      {useCaseOptions.map(useCase => (
                        <option key={useCase} value={useCase}>{useCase}</option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <Label htmlFor="team_size">Team Size</Label>
                    <select
                      id="team_size"
                      {...form.register('team_size')}
                      className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2"
                    >
                      <option value="">Select your team size (optional)</option>
                      {teamSizeOptions.map(size => (
                        <option key={size} value={size}>{size}</option>
                      ))}
                    </select>
                  </div>

                  <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                    <p className="text-sm text-green-800">
                      This information helps us provide relevant features and recommendations for your workflow.
                    </p>
                  </div>
                </div>
              )}

              {/* Step 4: Complete */}
              {currentStep === 3 && (
                <div className="text-center space-y-6">
                  <div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto">
                    <CheckCircle className="w-10 h-10 text-green-600" />
                  </div>
                  
                  <div>
                    <h3 className="text-xl font-semibold text-gray-900 mb-2">
                      Welcome to AgentScan, {form.getValues('full_name')}!
                    </h3>
                    <p className="text-gray-600">
                      You're all set up and ready to start scanning your repositories for security issues.
                    </p>
                  </div>

                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                    <h4 className="font-medium text-blue-900 mb-2">What's next?</h4>
                    <ul className="text-sm text-blue-800 space-y-1 text-left">
                      <li>• Connect your first repository</li>
                      <li>• Run your first security scan</li>
                      <li>• Explore the dashboard and findings</li>
                      <li>• Set up notifications and preferences</li>
                    </ul>
                  </div>
                </div>
              )}
            </form>

            {/* Navigation */}
            <div className="flex items-center justify-between mt-8 pt-6 border-t">
              <div className="flex items-center space-x-3">
                {!isFirstStep && (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handlePrevious}
                    disabled={isLoading}
                  >
                    <ArrowLeft className="w-4 h-4 mr-2" />
                    Previous
                  </Button>
                )}
                
                {currentStep < 2 && (
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={handleSkip}
                    disabled={isLoading}
                  >
                    Skip for now
                  </Button>
                )}
              </div>

              <div className="flex items-center space-x-3">
                <Badge variant="outline">
                  Step {currentStep + 1} of {steps.length}
                </Badge>
                
                <Button
                  type="button"
                  onClick={handleNext}
                  disabled={isLoading || (currentStep < 3 && !validateCurrentStep())}
                >
                  {isLoading ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Setting up...
                    </>
                  ) : isLastStep ? (
                    'Get Started'
                  ) : (
                    <>
                      Next
                      <ArrowRight className="w-4 h-4 ml-2" />
                    </>
                  )}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}