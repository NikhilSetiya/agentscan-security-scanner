-- AgentScan Initial Database Schema
-- This migration creates the core tables for the AgentScan application

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable Row Level Security
ALTER DATABASE postgres SET "app.jwt_secret" TO 'your-jwt-secret-here';

-- Users table (extends Supabase auth.users)
CREATE TABLE public.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    supabase_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    avatar_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Repositories table
CREATE TABLE public.repositories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    language VARCHAR(100),
    branch VARCHAR(255) DEFAULT 'main',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_scan_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, url)
);

-- Scans table
CREATE TABLE public.scans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    repository_id UUID NOT NULL REFERENCES public.repositories(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    findings_count INTEGER DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    branch VARCHAR(255) NOT NULL,
    commit_hash VARCHAR(255),
    commit_message TEXT,
    scan_type VARCHAR(50) DEFAULT 'full' CHECK (scan_type IN ('full', 'incremental')),
    triggered_by VARCHAR(255),
    agents TEXT[], -- Array of agent names used in scan
    duration INTERVAL,
    error_message TEXT
);

-- Findings table
CREATE TABLE public.findings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_id UUID NOT NULL REFERENCES public.scans(id) ON DELETE CASCADE,
    rule_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    file_path TEXT NOT NULL,
    line_number INTEGER,
    column_number INTEGER,
    tool VARCHAR(100) NOT NULL,
    confidence INTEGER DEFAULT 100 CHECK (confidence >= 0 AND confidence <= 100),
    status VARCHAR(50) DEFAULT 'open' CHECK (status IN ('open', 'ignored', 'fixed', 'false_positive')),
    code_snippet TEXT,
    fix_suggestion TEXT,
    references TEXT[], -- Array of reference URLs
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Finding suppressions table
CREATE TABLE public.finding_suppressions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    repository_id UUID REFERENCES public.repositories(id) ON DELETE CASCADE,
    rule_id VARCHAR(255) NOT NULL,
    file_path TEXT,
    reason TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID NOT NULL REFERENCES public.users(id)
);

-- User feedback table
CREATE TABLE public.user_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    finding_id UUID REFERENCES public.findings(id) ON DELETE CASCADE,
    feedback_type VARCHAR(50) NOT NULL CHECK (feedback_type IN ('false_positive', 'true_positive', 'suggestion')),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Secrets table for secure storage of application secrets
CREATE TABLE public.secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_users_supabase_id ON public.users(supabase_id);
CREATE INDEX idx_users_email ON public.users(email);
CREATE INDEX idx_repositories_user_id ON public.repositories(user_id);
CREATE INDEX idx_repositories_url ON public.repositories(url);
CREATE INDEX idx_scans_repository_id ON public.scans(repository_id);
CREATE INDEX idx_scans_user_id ON public.scans(user_id);
CREATE INDEX idx_scans_status ON public.scans(status);
CREATE INDEX idx_scans_started_at ON public.scans(started_at DESC);
CREATE INDEX idx_findings_scan_id ON public.findings(scan_id);
CREATE INDEX idx_findings_severity ON public.findings(severity);
CREATE INDEX idx_findings_status ON public.findings(status);
CREATE INDEX idx_findings_tool ON public.findings(tool);
CREATE INDEX idx_finding_suppressions_user_id ON public.finding_suppressions(user_id);
CREATE INDEX idx_finding_suppressions_rule_id ON public.finding_suppressions(rule_id);
CREATE INDEX idx_user_feedback_user_id ON public.user_feedback(user_id);
CREATE INDEX idx_user_feedback_finding_id ON public.user_feedback(finding_id);
CREATE INDEX idx_secrets_name ON public.secrets(name);

-- Row Level Security (RLS) Policies

-- Enable RLS on all tables
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.repositories ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.scans ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.findings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.finding_suppressions ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.secrets ENABLE ROW LEVEL SECURITY;

-- Users table policies
CREATE POLICY "Users can view their own profile" ON public.users
    FOR SELECT USING (supabase_id = auth.uid());

CREATE POLICY "Users can update their own profile" ON public.users
    FOR UPDATE USING (supabase_id = auth.uid());

CREATE POLICY "Users can insert their own profile" ON public.users
    FOR INSERT WITH CHECK (supabase_id = auth.uid());

-- Repositories table policies
CREATE POLICY "Users can view their own repositories" ON public.repositories
    FOR SELECT USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can insert their own repositories" ON public.repositories
    FOR INSERT WITH CHECK (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can update their own repositories" ON public.repositories
    FOR UPDATE USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can delete their own repositories" ON public.repositories
    FOR DELETE USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

-- Scans table policies
CREATE POLICY "Users can view their own scans" ON public.scans
    FOR SELECT USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can insert their own scans" ON public.scans
    FOR INSERT WITH CHECK (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can update their own scans" ON public.scans
    FOR UPDATE USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

-- Findings table policies
CREATE POLICY "Users can view findings from their scans" ON public.findings
    FOR SELECT USING (scan_id IN (SELECT id FROM public.scans WHERE user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid())));

CREATE POLICY "System can insert findings" ON public.findings
    FOR INSERT WITH CHECK (true); -- Allow system to insert findings

CREATE POLICY "Users can update findings from their scans" ON public.findings
    FOR UPDATE USING (scan_id IN (SELECT id FROM public.scans WHERE user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid())));

-- Finding suppressions table policies
CREATE POLICY "Users can view their own suppressions" ON public.finding_suppressions
    FOR SELECT USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can insert their own suppressions" ON public.finding_suppressions
    FOR INSERT WITH CHECK (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can update their own suppressions" ON public.finding_suppressions
    FOR UPDATE USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can delete their own suppressions" ON public.finding_suppressions
    FOR DELETE USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

-- User feedback table policies
CREATE POLICY "Users can view their own feedback" ON public.user_feedback
    FOR SELECT USING (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

CREATE POLICY "Users can insert their own feedback" ON public.user_feedback
    FOR INSERT WITH CHECK (user_id IN (SELECT id FROM public.users WHERE supabase_id = auth.uid()));

-- Secrets table policies (only service role can access)
CREATE POLICY "Only service role can access secrets" ON public.secrets
    FOR ALL USING (auth.role() = 'service_role');

-- Functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_findings_updated_at BEFORE UPDATE ON public.findings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically create user profile when auth user is created
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO public.users (supabase_id, email, name, avatar_url)
    VALUES (
        NEW.id,
        NEW.email,
        COALESCE(NEW.raw_user_meta_data->>'name', split_part(NEW.email, '@', 1)),
        NEW.raw_user_meta_data->>'avatar_url'
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Trigger to automatically create user profile
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();

-- Function to get dashboard statistics
CREATE OR REPLACE FUNCTION public.get_dashboard_stats(user_uuid UUID)
RETURNS JSON AS $$
DECLARE
    result JSON;
    user_internal_id UUID;
BEGIN
    -- Get internal user ID
    SELECT id INTO user_internal_id FROM public.users WHERE supabase_id = user_uuid;
    
    IF user_internal_id IS NULL THEN
        RETURN '{"error": "User not found"}'::JSON;
    END IF;

    SELECT json_build_object(
        'total_scans', (SELECT COUNT(*) FROM public.scans WHERE user_id = user_internal_id),
        'total_repositories', (SELECT COUNT(*) FROM public.repositories WHERE user_id = user_internal_id),
        'findings_by_severity', json_build_object(
            'critical', (SELECT COUNT(*) FROM public.findings f JOIN public.scans s ON f.scan_id = s.id WHERE s.user_id = user_internal_id AND f.severity = 'critical'),
            'high', (SELECT COUNT(*) FROM public.findings f JOIN public.scans s ON f.scan_id = s.id WHERE s.user_id = user_internal_id AND f.severity = 'high'),
            'medium', (SELECT COUNT(*) FROM public.findings f JOIN public.scans s ON f.scan_id = s.id WHERE s.user_id = user_internal_id AND f.severity = 'medium'),
            'low', (SELECT COUNT(*) FROM public.findings f JOIN public.scans s ON f.scan_id = s.id WHERE s.user_id = user_internal_id AND f.severity = 'low'),
            'info', (SELECT COUNT(*) FROM public.findings f JOIN public.scans s ON f.scan_id = s.id WHERE s.user_id = user_internal_id AND f.severity = 'info')
        ),
        'recent_scans', (
            SELECT COALESCE(json_agg(
                json_build_object(
                    'id', s.id,
                    'repository_id', s.repository_id,
                    'repository', json_build_object(
                        'id', r.id,
                        'name', r.name,
                        'url', r.url,
                        'language', r.language,
                        'branch', r.branch,
                        'created_at', r.created_at,
                        'last_scan_at', r.last_scan_at
                    ),
                    'status', s.status,
                    'progress', s.progress,
                    'findings_count', s.findings_count,
                    'started_at', s.started_at,
                    'completed_at', s.completed_at,
                    'duration', EXTRACT(EPOCH FROM s.duration)::INTEGER,
                    'branch', s.branch,
                    'commit', s.commit_hash,
                    'commit_message', s.commit_message,
                    'triggered_by', s.triggered_by,
                    'scan_type', s.scan_type
                )
            ), '[]'::json)
            FROM public.scans s
            JOIN public.repositories r ON s.repository_id = r.id
            WHERE s.user_id = user_internal_id
            ORDER BY s.started_at DESC
            LIMIT 10
        )
    ) INTO result;

    RETURN result;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;