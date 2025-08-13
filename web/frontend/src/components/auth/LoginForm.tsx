import React, { useState } from 'react';
import { Button } from '../ui/Button';
import { Card, CardHeader, CardTitle, CardContent } from '../ui/Card';
import { useAuth } from '../../contexts/AuthContext';
import { Eye, EyeOff, AlertCircle } from 'lucide-react';
import './LoginForm.css';

interface LoginFormProps {
  onSuccess?: () => void;
}

export const LoginForm: React.FC<LoginFormProps> = ({ onSuccess }) => {
  const { state, login, clearError } = useAuth();
  const [formData, setFormData] = useState({
    username: '',
    password: '',
  });
  const [showPassword, setShowPassword] = useState(false);
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear validation error when user starts typing
    if (validationErrors[name]) {
      setValidationErrors(prev => ({ ...prev, [name]: '' }));
    }
    
    // Clear API error when user starts typing
    if (state.error) {
      clearError();
    }
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    if (!formData.username.trim()) {
      errors.username = 'Username or email is required';
    } else if (formData.username.includes('@') && !/\S+@\S+\.\S+/.test(formData.username)) {
      errors.username = 'Please enter a valid email address';
    }

    if (!formData.password) {
      errors.password = 'Password is required';
    } else if (formData.password.length < 6) {
      errors.password = 'Password must be at least 6 characters';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    const success = await login(formData);
    if (success) {
      onSuccess?.();
    }
  };

  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
  };

  return (
    <div className="login-container">
      <Card className="login-card">
        <CardHeader>
          <CardTitle className="login-title">
            Sign in to AgentScan
          </CardTitle>
          <p className="login-subtitle">
            Enter your credentials to access your security dashboard
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

            <div className="form-group">
              <label htmlFor="username" className="form-label">
                Username or Email
              </label>
              <input
                id="username"
                name="username"
                type="text"
                value={formData.username}
                onChange={handleInputChange}
                className={`form-input ${validationErrors.username ? 'error' : ''}`}
                placeholder="Enter your username or email"
                disabled={state.isLoading}
                autoComplete="username"
              />
              {validationErrors.username && (
                <span className="error-text">{validationErrors.username}</span>
              )}
            </div>

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
                  autoComplete="current-password"
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

            <Button
              type="submit"
              variant="primary"
              size="lg"
              className="login-button"
              loading={state.isLoading}
              loadingText="Signing in..."
              disabled={state.isLoading}
            >
              Sign In
            </Button>

            <div className="login-footer">
              <p className="demo-credentials">
                <strong>Demo Credentials:</strong><br />
                Username: demo@agentscan.dev<br />
                Password: demo123
              </p>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
};