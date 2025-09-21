-- Initial QuantumLayer Factory schema
-- Creates tables for briefs, IR specs, runs, and gates

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table for storing user briefs (natural language specifications)
CREATE TABLE briefs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    text TEXT NOT NULL,
    user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Table for storing compiled intermediate representations
CREATE TABLE ir_specs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    brief_id UUID REFERENCES briefs(id) ON DELETE CASCADE,
    version VARCHAR(10) NOT NULL,
    spec JSONB NOT NULL,
    approved BOOLEAN DEFAULT FALSE,
    compiled_at TIMESTAMP DEFAULT NOW(),
    compiled_by VARCHAR(255)
);

-- Table for tracking factory runs (end-to-end generation sessions)
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ir_id UUID REFERENCES ir_specs(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    overlays TEXT[], -- Array of overlay names applied
    preview_url TEXT,
    capsule_url TEXT, -- Final packaged output
    tokens_used INTEGER DEFAULT 0,
    cost_usd DECIMAL(10,4) DEFAULT 0.00,
    temporal_workflow_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Table for tracking verification gate results
CREATE TABLE gates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID REFERENCES runs(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    report JSONB,
    execution_time_ms INTEGER,
    attempts INTEGER DEFAULT 1,
    last_attempt_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Table for storing agent-generated patches
CREATE TABLE patches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID REFERENCES runs(id) ON DELETE CASCADE,
    agent_name VARCHAR(50) NOT NULL,
    task_description TEXT,
    files TEXT[] NOT NULL, -- Array of file paths
    content TEXT NOT NULL, -- The unified diff content
    status VARCHAR(20) DEFAULT 'pending',
    applied_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Table for storing learned patterns and failures for RAG
CREATE TABLE knowledge_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category VARCHAR(50) NOT NULL, -- 'pattern', 'failure', 'repair'
    brief_context TEXT,
    technical_context JSONB,
    solution JSONB,
    embedding VECTOR(1536), -- For vector similarity search
    success_rate DECIMAL(5,2) DEFAULT 100.0,
    usage_count INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Table for overlay configurations
CREATE TABLE overlays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(50) NOT NULL, -- 'domain', 'capability', 'locale'
    version VARCHAR(20) DEFAULT '1.0.0',
    config JSONB NOT NULL,
    requires TEXT[], -- Array of dependency overlay names
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_briefs_created_at ON briefs(created_at);
CREATE INDEX idx_briefs_user_id ON briefs(user_id);

CREATE INDEX idx_ir_specs_brief_id ON ir_specs(brief_id);
CREATE INDEX idx_ir_specs_approved ON ir_specs(approved);

CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_created_at ON runs(created_at);
CREATE INDEX idx_runs_ir_id ON runs(ir_id);

CREATE INDEX idx_gates_run_id ON gates(run_id);
CREATE INDEX idx_gates_status ON gates(status);
CREATE INDEX idx_gates_name ON gates(name);

CREATE INDEX idx_patches_run_id ON patches(run_id);
CREATE INDEX idx_patches_status ON patches(status);
CREATE INDEX idx_patches_agent_name ON patches(agent_name);

CREATE INDEX idx_knowledge_category ON knowledge_entries(category);
CREATE INDEX idx_knowledge_updated_at ON knowledge_entries(updated_at);

CREATE INDEX idx_overlays_category ON overlays(category);
CREATE INDEX idx_overlays_active ON overlays(active);

-- Triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_briefs_updated_at BEFORE UPDATE ON briefs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_runs_updated_at BEFORE UPDATE ON runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_knowledge_entries_updated_at BEFORE UPDATE ON knowledge_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_overlays_updated_at BEFORE UPDATE ON overlays
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();