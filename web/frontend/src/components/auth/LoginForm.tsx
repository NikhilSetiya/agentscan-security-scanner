import React, { useState } from 'react';
import { Button } from '../ui/Button';
import { Card, CardHeader, CardTitle, CardContent } from '../ui/Card';
import { useAuth } from '../../contexts/AuthContext';
import { Eye, EyeOff, AlertCircle, CheckCircle } from 'lucide-react';
import './LoginForm.css';

type FormMode = 'signin' | 'signup' | 'reset';

interface LoginFormProps {
  onSuccess?: () => void;
}

export const LoginForm: React.FC<LoginFormProps> = ({ onSuccess }) => {
  const { state, signIn, signUp, resetPassword, clearError } = useAuth();
  const [mode, setMode] = useState<FormMode>('signin');
  const [formData, setFormData] = useState({
    email: '',
    password: '',
    name: '',
    confirmPassword: '',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});
  const [successMessage, setSuccessMessage] = useState<string>('');

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear validation error when user starts typing
    if (validationErrors[name]) {
      setValidationErrors(prev => ({ ...prev, [name]: '' }));
    }
    
    // Clear API error and success message when user starts typing
    if (state.error) {
      clearError();
    }
    if (successMessage) {
      setSuccessMessage('');
    }
  };

  const handleModeChange = (newMode: FormMode) => {
    setMode(newMode);
    setFormData({
      email: '',
      password: '',
      name: '',
      confirmPassword: '',
    });
    setValidationErrors({});
    setSuccessMessage('');
    if (state.error) {
      clearError();
    }
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    // Email validation (required for all modes)
    if (!formData.email.trim()) {
      errors.email = 'Email is required';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      errors.email = 'Please enter a valid email address';
    }

    // Password validation (not required for reset mode)
    if (mode !== 'reset') {
      if (!formData.password) {
        errors.password = 'Password is required';
      } else if (formData.password.length < 6) {
        errors.password = 'Password must be at least 6 characters';
      }
    }

    // Additional validation for signup mode
    if (mode === 'signup') {
      if (!formData.name.trim()) {
        errors.name = 'Name is required';
      }

      if (!formData.confirmPassword) {
        errors.confirmPassword = 'Please confirm your password';
      } else if (formData.password !== formData.confirmPassword) {
        errors.confirmPassword = 'Passwords do not match';
      }
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    let success = false;

    switch (mode) {
      case 'signin':
        success = await signIn({
          email: formData.email,
          password: formData.password,
        });
        if (success) {
          onSuccess?.();
        }
        break;

      case 'signup':
        success = await signUp({
          email: formData.email,
          password: formData.password,
          name: formData.name,
        });
        if (success) {
          setSuccessMessage('Account created successfully! Please check your email to verify your account before signing in.');
          setMode('signin');
        }
        break;

      case 'reset':
        success = await resetPassword({
          email: formData.email,
        });
        if (success) {
          setSuccessMessage('Password reset email sent! Please check your email for instructions.');
          setMode('signin');
        }
        break;
    }
  };

  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
  };

  const toggleConfirmPasswordVisibility = () => {
    setShowConfirmPassword(!showConfirmPassword);
  };

  const getTitle = () => {
    switch (mode) {
      case 'signin':
        return 'Sign in to AgentScan';
      case 'signup':
        return 'Create your AgentScan account';
      case 'reset':
        return 'Reset your password';
    }
  };

  const getSubtitle = () => {
    switch (mode) {
      case 'signin':
        return 'Enter your credentials to access your security dashboard';
      case 'signup':
        return 'Get started with AgentScan security scanning';
      case 'reset':
        return 'Enter your email to receive password reset instructions';
    }
  };

  const getButtonText = () => {
    switch (mode) {
      case 'signin':
        return 'Sign In';
      case 'signup':
        return 'Create Account';
      case 'reset':
        return 'Send Reset Email';
    }
  };

  const getLoadingText = () => {
    switch (mode) {
      case 'signin':
        return 'Signing in...';
      case 'signup':
        return 'Creating account...';
      case 'reset':
        return 'Sending email...';
    }
  };

  return (
    <div className="login-container">
      <Card className="login-card">
        <CardHeader>
          <CardTitle className="login-title">
            {getTitle()}
          </CardTitle>
          <p className="login-subtitle">
            {getSubtitle()}
          </p>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="login-form">
            {state.error && (
              <div className="error-banner">
                <AlertCircle size={16} />
                <span>{state.error}</span>
              </div>
            )}

            {successMessage && (
              <div className="success-banner">
                <CheckCircle size={16} />
                <span>{successMessage}</span>
              </div>
            )}

            <div className="form-group">
              <label htmlFor="email" className="form-label">
                Email
              </label>
              <input
                id="email"
                name="email"
                type="email"
                value={formData.email}
                onChange={handleInputChange}
                className={`form-input ${validationErrors.email ? 'error' : ''}`}
                placeholder="Enter your email address"
                disabled={state.isLoading}
                autoComplete="email"
              />
              {validationErrors.email && (
                <span className="error-text">{validationErrors.email}</span>
              )}
            </div>

            {mode === 'signup' && (
              <div className="form-group">
                <label htmlFor="name" className="form-label">
                  Full Name
                </label>
                <input
                  id="name"
                  name="name"
                  type="text"
                  value={formData.name}
                  onChange={handleInputChange}
                  className={`form-input ${validationErrors.name ? 'error' : ''}`}
                  placeholder="Enter your full name"
                  disabled={state.isLoading}
                  autoComplete="name"
                />
                {validationErrors.name && (
                  <span className="error-text">{validationErrors.name}</span>
                )}
              </div>
            )}

            {mode !== 'reset' && (
              <div className="form-group">
                <label htmlFor="password" className="form-label">
                  Password
                </label>
                <div className="password-input-container">
                  <input
                    id="password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    value={formData.password}
                    onChange={handleInputChange}
                    className={`form-input ${validationErrors.password ? 'error' : ''}`}
                    placeholder="Enter your password"
                    disabled={state.isLoading}
                    autoComplete={mode === 'signup' ? 'new-password' : 'current-password'}
                  />
                  <button
                    type="button"
                    onClick={togglePasswordVisibility}
                    className="password-toggle"
                    disabled={state.isLoading}
                    aria-label={showPassword ? 'Hide password' : 'Show password'}
                  >
                    {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
                {validationErrors.password && (
                  <span className="error-text">{validationErrors.password}</span>
                )}
              </div>
            )}

            {mode === 'signup' && (
              <div className="form-group">
                <label htmlFor="confirmPassword" className="form-label">
                  Confirm Password
                </label>
                <div className="password-input-container">
                  <input
                    id="confirmPassword"
                    name="confirmPassword"
                    type={showConfirmPassword ? 'text' : 'password'}
                    value={formData.confirmPassword}
                    onChange={handleInputChange}
                    className={`form-input ${validationErrors.confirmPassword ? 'error' : ''}`}
                    placeholder="Confirm your password"
                    disabled={state.isLoading}
                    autoComplete="new-password"
                  />
                  <button
                    type="button"
                    onClick={toggleConfirmPasswordVisibility}
                    className="password-toggle"
                    disabled={state.isLoading}
                    aria-label={showConfirmPassword ? 'Hide password' : 'Show password'}
                  >
                    {showConfirmPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
                {validationErrors.confirmPassword && (
                  <span className="error-text">{validationErrors.confirmPassword}</span>
                )}
              </div>
            )}

            <Button
              type="submit"
              variant="primary"
              size="lg"
              className="login-button"
              loading={state.isLoading}
              loadingText={getLoadingText()}
              disabled={state.isLoading}
            >
              {getButtonText()}
            </Button>

            <div className="login-footer">
              {mode === 'signin' && (
                <>
                  <button
                    type="button"
                    onClick={() => handleModeChange('reset')}
                    className="link-button"
                    disabled={state.isLoading}
                  >
                    Forgot your password?
                  </button>
                  <p className="auth-switch">
                    Don't have an account?{' '}
                    <button
                      type="button"
                      onClick={() => handleModeChange('signup')}
                      className="link-button"
                      disabled={state.isLoading}
                    >
                      Sign up
                    </button>
                  </p>
                </>
              )}

              {mode === 'signup' && (
                <p className="auth-switch">
                  Already have an account?{' '}
                  <button
                    type="button"
                    onClick={() => handleModeChange('signin')}
                    className="link-button"
                    disabled={state.isLoading}
                  >
                    Sign in
                  </button>
                </p>
              )}

              {mode === 'reset' && (
                <p className="auth-switch">
                  Remember your password?{' '}
                  <button
                    type="button"
                    onClick={() => handleModeChange('signin')}
                    className="link-button"
                    disabled={state.isLoading}
                  >
                    Sign in
                  </button>
                </p>
              )}
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
};