import Link from 'next/link'
import { ArrowRight, Play, Shield, Zap, Users } from 'lucide-react'

export function Hero() {
  return (
    <section className="relative bg-gradient-to-br from-primary-50 via-white to-primary-50 py-20 sm:py-32">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center">
          {/* Badge */}
          <div className="inline-flex items-center px-4 py-2 rounded-full bg-primary-100 text-primary-800 text-sm font-medium mb-8">
            <Shield className="h-4 w-4 mr-2" />
            Comprehensive Security Scanning Platform
          </div>

          {/* Main heading */}
          <h1 className="text-4xl sm:text-6xl font-bold text-gray-900 mb-6">
            Secure Your Code with{' '}
            <span className="text-primary-600">AgentScan</span>
          </h1>

          {/* Subtitle */}
          <p className="text-xl sm:text-2xl text-gray-600 max-w-4xl mx-auto mb-10">
            Multi-language security scanning with consensus-based results, 
            real-time feedback, and seamless CI/CD integration.
          </p>

          {/* CTA Buttons */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center mb-16">
            <Link
              href="/docs/getting-started/installation"
              className="btn btn-primary text-lg px-8 py-3"
            >
              Get Started
              <ArrowRight className="ml-2 h-5 w-5" />
            </Link>
            <Link
              href="/docs/examples/quick-demo"
              className="btn btn-outline text-lg px-8 py-3"
            >
              <Play className="mr-2 h-5 w-5" />
              View Demo
            </Link>
          </div>

          {/* Feature highlights */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-8 max-w-4xl mx-auto">
            <div className="flex flex-col items-center text-center">
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center mb-4">
                <Shield className="h-6 w-6 text-primary-600" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                Multi-Tool Consensus
              </h3>
              <p className="text-gray-600">
                Aggregate results from multiple security tools for higher confidence findings
              </p>
            </div>

            <div className="flex flex-col items-center text-center">
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center mb-4">
                <Zap className="h-6 w-6 text-primary-600" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                Real-Time Results
              </h3>
              <p className="text-gray-600">
                Get instant feedback with WebSocket-powered live scan progress and results
              </p>
            </div>

            <div className="flex flex-col items-center text-center">
              <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center mb-4">
                <Users className="h-6 w-6 text-primary-600" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                Team Collaboration
              </h3>
              <p className="text-gray-600">
                Role-based access control and collaborative finding management
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Background decoration */}
      <div className="absolute inset-0 -z-10 overflow-hidden">
        <div className="absolute left-1/2 top-0 -translate-x-1/2 -translate-y-1/2">
          <div className="w-[800px] h-[800px] rounded-full bg-gradient-to-br from-primary-100/50 to-transparent blur-3xl" />
        </div>
      </div>
    </section>
  )
}