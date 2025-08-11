import React, { useState, useEffect } from 'react';
import { ChevronRight, ChevronLeft, Check, Shield, Zap, Users, ArrowRight } from 'lucide-react';
import { Modal, ModalContent } from '../ui/Modal';
import { Button } from '../ui/Button';
import './OnboardingFlow.css';

export interface OnboardingStep {
  id: string;
  title: string;
  description: string;
  content: React.ReactNode;
  icon?: React.ReactNode;
  optional?: boolean;
}

export interface OnboardingFlowProps {
  isOpen: boolean;
  onClose: () => void;
  onComplete: () => void;
  steps: OnboardingStep[];
  showProgress?: boolean;
}

const defaultSteps: OnboardingStep[] = [
  {
    id: 'welcome',
    title: 'Welcome to AgentScan',
    description: 'Your intelligent security scanning platform',
    icon: <Shield size={48} />,
    content: (
      <div className="onboarding-welcome">
        <div className="welcome-features">
          <div className="welcome-feature">
            <div className="welcome-feature-icon">
              <Shield size={24} />
            </div>
            <div>
              <h4>Multi-Agent Scanning</h4>
              <p>Run multiple security tools simultaneously for comprehensive coverage</p>
            </div>
          </div>
          <div className="welcome-feature">
            <div className="welcome-feature-icon">
              <Zap size={24} />
            </div>
            <div>
              <h4>Reduced False Positives</h4>
              <p>Intelligent consensus reduces false positives by up to 80%</p>
            </div>
          </div>
          <div className="welcome-feature">
            <div className="welcome-feature-icon">
              <Users size={24} />
            </div>
            <div>
              <h4>Team Collaboration</h4>
              <p>Share results and collaborate on security findings</p>
            </div>
          </div>
        </div>
      </div>
    ),
  },
  {
    id: 'first-scan',
    title: 'Run Your First Scan',
    description: 'Let\'s scan your first repository',
    icon: <Zap size={48} />,
    content: (
      <div className="onboarding-scan">
        <div className="scan-steps">
          <div className="scan-step">
            <div className="scan-step-number">1</div>
            <div>
              <h4>Connect Repository</h4>
              <p>Connect your GitHub, GitLab, or Bitbucket repository</p>
            </div>
          </div>
          <div className="scan-step">
            <div className="scan-step-number">2</div>
            <div>
              <h4>Configure Scan</h4>
              <p>Choose which security tools to run and set preferences</p>
            </div>
          </div>
          <div className="scan-step">
            <div className="scan-step-number">3</div>
            <div>
              <h4>Review Results</h4>
              <p>Get actionable security findings with confidence scores</p>
            </div>
          </div>
        </div>
        <div className="scan-cta">
          <Button variant="primary" size="lg">
            <ArrowRight size={16} />
            Start First Scan
          </Button>
        </div>
      </div>
    ),
  },
  {
    id: 'keyboard-shortcuts',
    title: 'Keyboard Shortcuts',
    description: 'Work faster with keyboard shortcuts',
    icon: <span className="keyboard-icon">⌨️</span>,
    optional: true,
    content: (
      <div className="onboarding-shortcuts">
        <div className="shortcuts-preview">
          <div className="shortcut-preview">
            <kbd>/</kbd>
            <span>Focus search</span>
          </div>
          <div className="shortcut-preview">
            <kbd>Ctrl</kbd> + <kbd>K</kbd>
            <span>Command palette</span>
          </div>
          <div className="shortcut-preview">
            <kbd>Ctrl</kbd> + <kbd>N</kbd>
            <span>New scan</span>
          </div>
          <div className="shortcut-preview">
            <kbd>?</kbd>
            <span>Show all shortcuts</span>
          </div>
        </div>
        <p className="shortcuts-note">
          Press <kbd>?</kbd> anytime to see all available shortcuts
        </p>
      </div>
    ),
  },
];

export const OnboardingFlow: React.FC<OnboardingFlowProps> = ({
  isOpen,
  onClose,
  onComplete,
  steps = defaultSteps,
  showProgress = true,
}) => {
  const [currentStep, setCurrentStep] = useState(0);
  const [completedSteps, setCompletedSteps] = useState<Set<number>>(new Set());

  const handleNext = () => {
    setCompletedSteps(prev => new Set([...prev, currentStep]));
    
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1);
    } else {
      onComplete();
      onClose();
    }
  };

  const handlePrevious = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleSkip = () => {
    onComplete();
    onClose();
  };

  const handleStepClick = (stepIndex: number) => {
    setCurrentStep(stepIndex);
  };

  const currentStepData = steps[currentStep];
  const isLastStep = currentStep === steps.length - 1;
  const isFirstStep = currentStep === 0;

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!isOpen) return;

      switch (event.key) {
        case 'ArrowRight':
        case 'Enter':
          if (!isLastStep) {
            event.preventDefault();
            handleNext();
          }
          break;
        case 'ArrowLeft':
          if (!isFirstStep) {
            event.preventDefault();
            handlePrevious();
          }
          break;
        case 'Escape':
          event.preventDefault();
          handleSkip();
          break;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, currentStep, isLastStep, isFirstStep]);

  if (!isOpen) return null;

  return (
    <Modal isOpen={isOpen} onClose={handleSkip} className="onboarding-modal">
      <ModalContent>
        <div className="onboarding-container">
          {/* Progress indicator */}
          {showProgress && (
            <div className="onboarding-progress">
              <div className="progress-steps">
                {steps.map((step, index) => (
                  <button
                    key={step.id}
                    className={`progress-step ${
                      index === currentStep ? 'active' : ''
                    } ${
                      completedSteps.has(index) ? 'completed' : ''
                    }`}
                    onClick={() => handleStepClick(index)}
                    aria-label={`Go to step ${index + 1}: ${step.title}`}
                  >
                    {completedSteps.has(index) ? (
                      <Check size={16} />
                    ) : (
                      <span>{index + 1}</span>
                    )}
                  </button>
                ))}
              </div>
              <div className="progress-bar">
                <div
                  className="progress-fill"
                  style={{
                    width: `${((currentStep + 1) / steps.length) * 100}%`,
                  }}
                />
              </div>
            </div>
          )}

          {/* Step content */}
          <div className="onboarding-step">
            <div className="step-header">
              {currentStepData.icon && (
                <div className="step-icon">{currentStepData.icon}</div>
              )}
              <div className="step-text">
                <h2 className="step-title">{currentStepData.title}</h2>
                <p className="step-description">{currentStepData.description}</p>
              </div>
            </div>

            <div className="step-content">
              {currentStepData.content}
            </div>
          </div>

          {/* Navigation */}
          <div className="onboarding-navigation">
            <div className="nav-left">
              {!isFirstStep && (
                <Button
                  variant="ghost"
                  onClick={handlePrevious}
                  size="md"
                >
                  <ChevronLeft size={16} />
                  Previous
                </Button>
              )}
            </div>

            <div className="nav-center">
              <span className="step-counter">
                {currentStep + 1} of {steps.length}
              </span>
            </div>

            <div className="nav-right">
              {currentStepData.optional && !isLastStep && (
                <Button
                  variant="ghost"
                  onClick={handleNext}
                  size="md"
                >
                  Skip
                </Button>
              )}
              <Button
                variant="primary"
                onClick={isLastStep ? () => { onComplete(); onClose(); } : handleNext}
                size="md"
              >
                {isLastStep ? 'Get Started' : 'Next'}
                {!isLastStep && <ChevronRight size={16} />}
              </Button>
            </div>
          </div>

          {/* Skip all option */}
          <div className="onboarding-skip">
            <button
              className="skip-button"
              onClick={handleSkip}
              aria-label="Skip onboarding"
            >
              Skip onboarding
            </button>
          </div>
        </div>
      </ModalContent>
    </Modal>
  );
};

// Hook to manage onboarding state
export const useOnboarding = () => {
  const [isOnboardingOpen, setIsOnboardingOpen] = useState(false);
  const [hasCompletedOnboarding, setHasCompletedOnboarding] = useState(false);

  useEffect(() => {
    // Check if user has completed onboarding
    const completed = localStorage.getItem('agentscan-onboarding-completed');
    if (completed === 'true') {
      setHasCompletedOnboarding(true);
    } else {
      // Show onboarding for new users
      setIsOnboardingOpen(true);
    }
  }, []);

  const completeOnboarding = () => {
    localStorage.setItem('agentscan-onboarding-completed', 'true');
    setHasCompletedOnboarding(true);
    setIsOnboardingOpen(false);
  };

  const resetOnboarding = () => {
    localStorage.removeItem('agentscan-onboarding-completed');
    setHasCompletedOnboarding(false);
    setIsOnboardingOpen(true);
  };

  return {
    isOnboardingOpen,
    hasCompletedOnboarding,
    completeOnboarding,
    resetOnboarding,
    openOnboarding: () => setIsOnboardingOpen(true),
    closeOnboarding: () => setIsOnboardingOpen(false),
  };
};