import { Copy, Terminal, CheckCircle } from 'lucide-react'

export function QuickStart() {
  const steps = [
    {
      title: 'Install AgentScan',
      description: 'Get started with Docker or download the binary',
      code: 'docker run -d -p 8080:8080 agentscan/agentscan:latest',
    },
    {
      title: 'Add Repository',
      description: 'Connect your first repository for scanning',
      code: 'curl -X POST http://localhost:8080/api/v1/repositories \\\n  -H "Authorization: Bearer $TOKEN" \\\n  -d \'{"name": "my-app", "url": "https://github.com/user/repo"}\'',
    },
    {
      title: 'Start Scanning',
      description: 'Run your first security scan',
      code: 'curl -X POST http://localhost:8080/api/v1/scans \\\n  -H "Authorization: Bearer $TOKEN" \\\n  -d \'{"repository_id": "repo-123", "agents": ["semgrep", "eslint"]}\'',
    },
  ]

  return (
    <section className="py-16 bg-gray-900">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-12">
          <h2 className="text-3xl font-bold text-white mb-4">
            Get Started in Minutes
          </h2>
          <p className="text-xl text-gray-300">
            Follow these simple steps to start scanning your code for security vulnerabilities
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {steps.map((step, index) => (
            <div key={index} className="relative">
              {/* Step number */}
              <div className="flex items-center mb-4">
                <div className="w-8 h-8 bg-primary-600 rounded-full flex items-center justify-center text-white font-semibold text-sm mr-3">
                  {index + 1}
                </div>
                <h3 className="text-xl font-semibold text-white">
                  {step.title}
                </h3>
              </div>

              {/* Description */}
              <p className="text-gray-300 mb-4">
                {step.description}
              </p>

              {/* Code block */}
              <div className="relative bg-gray-800 rounded-lg p-4 border border-gray-700">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <Terminal className="h-4 w-4 text-gray-400" />
                    <span className="text-sm text-gray-400">Terminal</span>
                  </div>
                  <button className="p-1 text-gray-400 hover:text-white transition-colors duration-200">
                    <Copy className="h-4 w-4" />
                  </button>
                </div>
                <pre className="text-sm text-gray-100 overflow-x-auto">
                  <code>{step.code}</code>
                </pre>
              </div>

              {/* Connector line */}
              {index < steps.length - 1 && (
                <div className="hidden lg:block absolute top-4 left-full w-8 h-0.5 bg-gray-700 transform translate-x-4" />
              )}
            </div>
          ))}
        </div>

        {/* Success message */}
        <div className="mt-12 text-center">
          <div className="inline-flex items-center px-6 py-3 bg-success-900/20 border border-success-700 rounded-lg text-success-300">
            <CheckCircle className="h-5 w-5 mr-2" />
            <span className="font-medium">
              That's it! Your security scanning platform is ready to use.
            </span>
          </div>
        </div>
      </div>
    </section>
  )
}