-- Initial schema for AgentScan
-- This migration creates the core database structure

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types for better type safety
CREATE TYPE scan_status AS ENUM ('queued', 'running', 'completed', 'failed', 'cancelled');
CREATE TYPE finding_severity AS ENUM ('high', 'medium', 'low', 'info');
CREATE TYPE finding_status AS ENUM ('open', 'fixed', 'ignored', 'false_positive');
CREATE TYPE user_role AS ENUM ('owner', 'admin', 'member');
CREATE TYPE scan_type AS ENUM ('full', 'incremental', 'ide');

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(500),
    github_id INTEGER UNIQUE,
    gitlab_id INTEGER UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organization members table
CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role user_role NOT NULL DEFAULT 'member',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

-- Repositories table
CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL,
    provider VARCHAR(50) NOT NULL, -- github, gitlab, bitbucket
    provider_id VARCHAR(100) NOT NULL,
    default_branch VARCHAR(100) DEFAULT 'main',
    languages JSONB DEFAULT '[]',
    settings JSONB DEFAULT '{}',
    last_scan_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, provider_id)
);

-- Scan jobs table
CREATE TABLE scan_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    branch VARCHAR(100) NOT NULL,
    commit_sha VARCHAR(40) NOT NULL,
    scan_type scan_type NOT NULL DEFAULT 'full',
    priority INTEGER DEFAULT 5,
    status scan_status NOT NULL DEFAULT 'queued',
    agents_requested JSONB DEFAULT '[]',
    agents_completed JSONB DEFAULT '[]',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Scan results table
CREATE TABLE scan_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_job_id UUID REFERENCES scan_jobs(id) ON DELETE CASCADE,
    agent_name VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    findings_count INTEGER DEFAULT 0,
    duration_ms INTEGER,
    error_message TEXT,
    raw_output JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Findings table
CREATE TABLE findings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_result_id UUID REFERENCES scan_results(id) ON DELETE CASCADE,
    scan_job_id UUID REFERENCES scan_jobs(id) ON DELETE CASCADE,
    tool VARCHAR(100) NOT NULL,
    rule_id VARCHAR(200) NOT NULL,
    severity finding_severity NOT NULL,
    category VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    file_path VARCHAR(1000) NOT NULL,
    line_number INTEGER,
    column_number INTEGER,
    code_snippet TEXT,
    confidence DECIMAL(3,2) DEFAULT 0.5,
    consensus_score DECIMAL(3,2),
    status finding_status DEFAULT 'open',
    fix_suggestion JSONB,
    references JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User feedback table for ML training
CREATE TABLE user_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    finding_id UUID REFERENCES findings(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL, -- fixed, ignored, false_positive, confirmed
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(finding_id, user_id)
);

-- Create function for updating updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_organizations_updated_at 
    BEFORE UPDATE ON organizations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_repositories_updated_at 
    BEFORE UPDATE ON repositories 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scan_jobs_updated_at 
    BEFORE UPDATE ON scan_jobs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_findings_updated_at 
    BEFORE UPDATE ON findings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();