import Link from 'next/link'
import { ArrowRight } from 'lucide-react'

interface FeatureCardProps {
  icon: React.ReactNode
  title: string
  description: string
  href: string
}

export function FeatureCard({ icon, title, description, href }: FeatureCardProps) {
  return (
    <Link href={href} className="group">
      <div className="card p-6 h-full transition-all duration-200 group-hover:shadow-lg group-hover:border-primary-200">
        <div className="flex items-center mb-4">
          <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center mr-4 group-hover:bg-primary-200 transition-colors duration-200">
            {icon}
          </div>
          <h3 className="text-xl font-semibold text-gray-900 group-hover:text-primary-600 transition-colors duration-200">
            {title}
          </h3>
        </div>
        <p className="text-gray-600 mb-4 flex-1">
          {description}
        </p>
        <div className="flex items-center text-primary-600 font-medium group-hover:text-primary-700 transition-colors duration-200">
          Learn more
          <ArrowRight className="ml-1 h-4 w-4 group-hover:translate-x-1 transition-transform duration-200" />
        </div>
      </div>
    </Link>
  )
}