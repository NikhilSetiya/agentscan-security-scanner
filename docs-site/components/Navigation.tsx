'use client'

import { useState } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Menu, X, Search, Github, ExternalLink } from 'lucide-react'
import { clsx } from 'clsx'

const navigation = [
  { name: 'Documentation', href: '/docs' },
  { name: 'API Reference', href: '/docs/api' },
  { name: 'Integrations', href: '/docs/integrations' },
  { name: 'Examples', href: '/docs/examples' },
  { name: 'Support', href: '/support' },
]

export function Navigation() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const pathname = usePathname()

  return (
    <nav className="bg-white border-b border-gray-200 sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          {/* Logo */}
          <div className="flex items-center">
            <Link href="/" className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">AS</span>
              </div>
              <span className="text-xl font-semibold text-gray-900">
                AgentScan
              </span>
            </Link>
          </div>

          {/* Desktop Navigation */}
          <div className="hidden md:flex items-center space-x-8">
            {navigation.map((item) => (
              <Link
                key={item.name}
                href={item.href}
                className={clsx(
                  'text-sm font-medium transition-colors duration-200',
                  pathname.startsWith(item.href)
                    ? 'text-primary-600'
                    : 'text-gray-700 hover:text-primary-600'
                )}
              >
                {item.name}
              </Link>
            ))}
          </div>

          {/* Right side actions */}
          <div className="hidden md:flex items-center space-x-4">
            {/* Search */}
            <button className="p-2 text-gray-400 hover:text-gray-600 transition-colors duration-200">
              <Search className="h-5 w-5" />
              <span className="sr-only">Search</span>
            </button>

            {/* GitHub */}
            <a
              href="https://github.com/agentscan/agentscan"
              target="_blank"
              rel="noopener noreferrer"
              className="p-2 text-gray-400 hover:text-gray-600 transition-colors duration-200"
            >
              <Github className="h-5 w-5" />
              <span className="sr-only">GitHub</span>
            </a>

            {/* Try AgentScan */}
            <Link
              href="https://app.agentscan.dev"
              className="btn btn-primary text-sm"
            >
              Try AgentScan
              <ExternalLink className="ml-1 h-3 w-3" />
            </Link>
          </div>

          {/* Mobile menu button */}
          <div className="md:hidden">
            <button
              type="button"
              className="p-2 text-gray-400 hover:text-gray-600"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            >
              {mobileMenuOpen ? (
                <X className="h-6 w-6" />
              ) : (
                <Menu className="h-6 w-6" />
              )}
              <span className="sr-only">Toggle menu</span>
            </button>
          </div>
        </div>
      </div>

      {/* Mobile menu */}
      {mobileMenuOpen && (
        <div className="md:hidden border-t border-gray-200 bg-white">
          <div className="px-4 py-2 space-y-1">
            {navigation.map((item) => (
              <Link
                key={item.name}
                href={item.href}
                className={clsx(
                  'block px-3 py-2 text-base font-medium rounded-md transition-colors duration-200',
                  pathname.startsWith(item.href)
                    ? 'text-primary-600 bg-primary-50'
                    : 'text-gray-700 hover:text-primary-600 hover:bg-gray-50'
                )}
                onClick={() => setMobileMenuOpen(false)}
              >
                {item.name}
              </Link>
            ))}
            
            <div className="pt-4 pb-2 border-t border-gray-200 mt-4">
              <div className="flex items-center space-x-4 px-3">
                <button className="p-2 text-gray-400 hover:text-gray-600">
                  <Search className="h-5 w-5" />
                </button>
                <a
                  href="https://github.com/agentscan/agentscan"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="p-2 text-gray-400 hover:text-gray-600"
                >
                  <Github className="h-5 w-5" />
                </a>
                <Link
                  href="https://app.agentscan.dev"
                  className="btn btn-primary text-sm flex-1 justify-center"
                >
                  Try AgentScan
                  <ExternalLink className="ml-1 h-3 w-3" />
                </Link>
              </div>
            </div>
          </div>
        </div>
      )}
    </nav>
  )
}