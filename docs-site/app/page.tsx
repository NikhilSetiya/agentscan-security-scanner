import Link from 'next/link'
import { ArrowRight, Shield, Zap, Users, Code, BookOpen, Rocket } from 'lucide-react'
import { Hero } from '@/components/Hero'
import { FeatureCard } from '@/components/FeatureCard'
import { QuickStart } from '@/components/QuickStart'

export default function HomePage() {
  return (
    <div className="min-h-screen">
      {/* Hero Section */}
      <Hero />

      {/* Features Section */}
      <section className="py-16 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold text-gray-900 mb-4">
              Comprehensive Security Scanning
            </h2>
            <p className="text-xl text-gray-600 max-w-3xl mx-auto">
              AgentScan orchestrates multiple security tools to provide unified vulnerability 
              assessment across different programming languages and frameworks.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            <FeatureCard
              icon={<Shield className="h-8 w-8 text-primary-600" />}
              title="Multi-Language SAST"
              description="Static analysis for JavaScript, Python, Go, Java, and more with consensus-based results."
              href="/docs/scanning/sast"
            />
            <FeatureCard
              icon={<Zap className="h-8 w-8 text-primary-600" />}
              title="Real-Time Scanning"
              description="Get instant feedback with WebSocket-powered real-time scan progress and results."
              href="/docs/scanning/real-time"
            />
            <FeatureCard
              icon={<Users className="h-8 w-8 text-primary-600" />}
              title="Team Collaboration"
              description="Role-based access control, finding management, and team-wide security insights."
              href="/docs/collaboration"
            />
            <FeatureCard
              icon={<Code className="h-8 w-8 text-primary-600" />}
              title="CI/CD Integration"
              description="Seamless integration with GitHub, GitLab, Jenkins, and other CI/CD platforms."
              href="/docs/integrations"
            />
            <FeatureCard
              icon={<BookOpen className="h-8 w-8 text-primary-600" />}
              title="Comprehensive API"
              description="Full REST API with OpenAPI specification for custom integrations and automation."
              href="/docs/api"
            />
            <FeatureCard
              icon={<Rocket className="h-8 w-8 text-primary-600" />}
              title="Easy Deployment"
              description="Deploy with Docker, Kubernetes, or cloud platforms with infrastructure as code."
              href="/docs/deployment"
            />
          </div>
        </div>
      </section>

      {/* Quick Start Section */}
      <QuickStart />

      {/* Documentation Sections */}
      <section className="py-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <h2 className="text-3xl font-bold text-gray-900 mb-4">
              Explore the Documentation
            </h2>
            <p className="text-xl text-gray-600">
              Everything you need to get started with AgentScan
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            <div className="card p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Getting Started
              </h3>
              <p className="text-gray-600 mb-4">
                Learn the basics of AgentScan, from installation to your first scan.
              </p>
              <div className="space-y-2">
                <Link href="/docs/getting-started/installation" className="block text-primary-600 hover:text-primary-700">
                  Installation Guide
                </Link>
                <Link href="/docs/getting-started/quick-start" className="block text-primary-600 hover:text-primary-700">
                  Quick Start Tutorial
                </Link>
                <Link href="/docs/getting-started/concepts" className="block text-primary-600 hover:text-primary-700">
                  Core Concepts
                </Link>
              </div>
            </div>

            <div className="card p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                API Reference
              </h3>
              <p className="text-gray-600 mb-4">
                Complete API documentation with examples and interactive explorer.
              </p>
              <div className="space-y-2">
                <Link href="/docs/api/authentication" className="block text-primary-600 hover:text-primary-700">
                  Authentication
                </Link>
                <Link href="/docs/api/repositories" className="block text-primary-600 hover:text-primary-700">
                  Repository Management
                </Link>
                <Link href="/docs/api/scanning" className="block text-primary-600 hover:text-primary-700">
                  Scanning Operations
                </Link>
              </div>
            </div>

            <div className="card p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Integrations
              </h3>
              <p className="text-gray-600 mb-4">
                Connect AgentScan with your existing development workflow.
              </p>
              <div className="space-y-2">
                <Link href="/docs/integrations/github" className="block text-primary-600 hover:text-primary-700">
                  GitHub Integration
                </Link>
                <Link href="/docs/integrations/vscode" className="block text-primary-600 hover:text-primary-700">
                  VS Code Extension
                </Link>
                <Link href="/docs/integrations/ci-cd" className="block text-primary-600 hover:text-primary-700">
                  CI/CD Pipelines
                </Link>
              </div>
            </div>

            <div className="card p-6">
              <h3 className="text-xl font-semibold text-gray-900 mb-3">
                Advanced Topics
              </h3>
              <p className="text-gray-600 mb-4">
                Deep dive into advanced features and customization options.
              </p>
              <div className="space-y-2">
                <Link href="/docs/advanced/custom-agents" className="block text-primary-600 hover:text-primary-700">
                  Custom Security Agents
                </Link>
                <Link href="/docs/advanced/consensus-engine" className="block text-primary-600 hover:text-primary-700">
                  Consensus Engine
                </Link>
                <Link href="/docs/advanced/deployment" className="block text-primary-600 hover:text-primary-700">
                  Production Deployment
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-16 bg-primary-600">
        <div className="max-w-4xl mx-auto text-center px-4 sm:px-6 lg:px-8">
          <h2 className="text-3xl font-bold text-white mb-4">
            Ready to Secure Your Code?
          </h2>
          <p className="text-xl text-primary-100 mb-8">
            Get started with AgentScan today and experience comprehensive security scanning.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Link
              href="/docs/getting-started/installation"
              className="btn btn-primary bg-white text-primary-600 hover:bg-gray-100"
            >
              Get Started
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
            <Link
              href="/docs/api"
              className="btn btn-outline border-white text-white hover:bg-white hover:text-primary-600"
            >
              View API Docs
            </Link>
          </div>
        </div>
      </section>
    </div>
  )
}