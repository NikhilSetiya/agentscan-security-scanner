/**
 * Supabase Client Configuration
 * Provides centralized Supabase client setup for authentication and data access
 */

import { createClient } from '@supabase/supabase-js'

// Supabase configuration from environment variables
const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY

if (!supabaseUrl || !supabaseAnonKey) {
  throw new Error('Missing Supabase environment variables. Please check VITE_SUPABASE_URL and VITE_SUPABASE_ANON_KEY')
}

// Create Supabase client with authentication configuration
export const supabase = createClient(supabaseUrl, supabaseAnonKey, {
  auth: {
    autoRefreshToken: true,
    persistSession: true,
    detectSessionInUrl: true,
    flowType: 'pkce'
  },
  global: {
    headers: {
      'X-Client-Info': 'agentscan-web@1.0.0'
    }
  }
})

// Database types for type safety
export interface Database {
  public: {
    Tables: {
      users: {
        Row: {
          id: string
          supabase_id: string
          email: string
          name: string | null
          avatar_url: string | null
          created_at: string
          updated_at: string
        }
        Insert: {
          id?: string
          supabase_id: string
          email: string
          name?: string | null
          avatar_url?: string | null
          created_at?: string
          updated_at?: string
        }
        Update: {
          id?: string
          supabase_id?: string
          email?: string
          name?: string | null
          avatar_url?: string | null
          created_at?: string
          updated_at?: string
        }
      }
      repositories: {
        Row: {
          id: string
          user_id: string
          name: string
          url: string
          language: string | null
          branch: string
          created_at: string
          last_scan_at: string | null
        }
        Insert: {
          id?: string
          user_id: string
          name: string
          url: string
          language?: string | null
          branch?: string
          created_at?: string
          last_scan_at?: string | null
        }
        Update: {
          id?: string
          user_id?: string
          name?: string
          url?: string
          language?: string | null
          branch?: string
          created_at?: string
          last_scan_at?: string | null
        }
      }
      scans: {
        Row: {
          id: string
          repository_id: string
          user_id: string
          status: string
          progress: number
          findings_count: number
          started_at: string
          completed_at: string | null
          branch: string
          commit_hash: string | null
          scan_type: string
        }
        Insert: {
          id?: string
          repository_id: string
          user_id: string
          status: string
          progress?: number
          findings_count?: number
          started_at?: string
          completed_at?: string | null
          branch: string
          commit_hash?: string | null
          scan_type?: string
        }
        Update: {
          id?: string
          repository_id?: string
          user_id?: string
          status?: string
          progress?: number
          findings_count?: number
          started_at?: string
          completed_at?: string | null
          branch?: string
          commit_hash?: string | null
          scan_type?: string
        }
      }
    }
  }
}

// Type-safe Supabase client
export type SupabaseClient = typeof supabase