import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Navigation } from '@/components/Navigation'
import { Footer } from '@/components/Footer'

const inter = Inter({ 
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-inter',
})

export const metadata: Metadata = {
  title: 'AgentScan Documentation',
  description: 'Comprehensive security scanning platform documentation',
  keywords: ['security', 'scanning', 'SAST', 'vulnerability', 'documentation'],
  authors: [{ name: 'AgentScan Team' }],
  openGraph: {
    title: 'AgentScan Documentation',
    description: 'Comprehensive security scanning platform documentation',
    type: 'website',
    url: 'https://docs.agentscan.dev',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AgentScan Documentation',
    description: 'Comprehensive security scanning platform documentation',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" className={inter.variable}>
      <body className="font-sans antialiased bg-white text-gray-900">
        <div className="min-h-screen flex flex-col">
          <Navigation />
          <main className="flex-1">
            {children}
          </main>
          <Footer />
        </div>
      </body>
    </html>
  )
}