-- +goose Up
-- Migration: Create RAG (Retrieval Augmented Generation) tables
-- Description: Creates tables for schema embeddings, KPI definitions, business glossary, and query context
-- Version: 004
-- Date: 2024-01-16

-- Enable pgvector extension if not already enabled
CREATE EXTENSION IF NOT EXISTS vector;

-- Create schema_embeddings table for storing vector embeddings of schema elements
CREATE TABLE IF NOT EXISTS schema_embeddings (
    id SERIAL PRIMARY KEY,
    data_source_id INTEGER NOT NULL,
    schema_id INTEGER NOT NULL,
    element_type VARCHAR(50) NOT NULL, -- 'table', 'column', 'kpi', 'glossary'
    element_name VARCHAR(255) NOT NULL,
    content TEXT, -- The text content that was embedded
    embedding vector(1536), -- OpenAI ada-002 embedding size
    metadata JSONB, -- Additional metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for schema_embeddings
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_data_source_id ON schema_embeddings(data_source_id);
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_schema_id ON schema_embeddings(schema_id);
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_element_type ON schema_embeddings(element_type);
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_element_name ON schema_embeddings(element_name);
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_deleted_at ON schema_embeddings(deleted_at);

-- Create vector similarity index using HNSW (Hierarchical Navigable Small World)
-- This index enables fast approximate nearest neighbor search
CREATE INDEX IF NOT EXISTS idx_schema_embeddings_embedding_cosine 
    ON schema_embeddings USING hnsw (embedding vector_cosine_ops);

-- Create KPI definitions table
CREATE TABLE IF NOT EXISTS kpi_definitions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200),
    description TEXT,
    formula TEXT, -- SQL formula or calculation
    category VARCHAR(50), -- revenue, marketing, operations, etc.
    unit VARCHAR(20), -- currency, percentage, count, etc.
    grain VARCHAR(20), -- daily, weekly, monthly, etc.
    filters JSONB, -- Default filters
    tags JSONB, -- Tags for categorization
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT unique_user_kpi_name UNIQUE(user_id, name, deleted_at)
);

-- Create indexes for kpi_definitions
CREATE INDEX IF NOT EXISTS idx_kpi_definitions_user_id ON kpi_definitions(user_id);
CREATE INDEX IF NOT EXISTS idx_kpi_definitions_category ON kpi_definitions(category);
CREATE INDEX IF NOT EXISTS idx_kpi_definitions_is_active ON kpi_definitions(is_active);
CREATE INDEX IF NOT EXISTS idx_kpi_definitions_deleted_at ON kpi_definitions(deleted_at);

-- Create business glossary table
CREATE TABLE IF NOT EXISTS business_glossaries (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    term VARCHAR(100) NOT NULL,
    definition TEXT NOT NULL,
    synonyms JSONB, -- Alternative terms
    category VARCHAR(50), -- business, technical, domain-specific
    domain VARCHAR(50), -- finance, marketing, operations, etc.
    examples JSONB, -- Usage examples
    related_terms JSONB, -- Related glossary terms
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT unique_user_term UNIQUE(user_id, term, deleted_at)
);

-- Create indexes for business_glossaries
CREATE INDEX IF NOT EXISTS idx_business_glossaries_user_id ON business_glossaries(user_id);
CREATE INDEX IF NOT EXISTS idx_business_glossaries_category ON business_glossaries(category);
CREATE INDEX IF NOT EXISTS idx_business_glossaries_domain ON business_glossaries(domain);
CREATE INDEX IF NOT EXISTS idx_business_glossaries_is_active ON business_glossaries(is_active);
CREATE INDEX IF NOT EXISTS idx_business_glossaries_deleted_at ON business_glossaries(deleted_at);

-- Create query context table for storing NL2SQL query context
CREATE TABLE IF NOT EXISTS rag_query_contexts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    data_source_id INTEGER NOT NULL,
    query TEXT NOT NULL,
    context JSONB, -- Retrieved context from RAG
    embedding vector(1536), -- Query embedding
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for rag_query_contexts
CREATE INDEX IF NOT EXISTS idx_rag_query_contexts_user_id ON rag_query_contexts(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_query_contexts_data_source_id ON rag_query_contexts(data_source_id);
CREATE INDEX IF NOT EXISTS idx_rag_query_contexts_created_at ON rag_query_contexts(created_at);

-- Create vector similarity index for query embeddings
CREATE INDEX IF NOT EXISTS idx_rag_query_contexts_embedding_cosine 
    ON rag_query_contexts USING hnsw (embedding vector_cosine_ops);

-- Note: Foreign key constraints will be added manually after confirming table existence

-- Create updated_at trigger function if it doesn't exist
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $func$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$func$ language 'plpgsql';

-- Create triggers for updated_at columns
DROP TRIGGER IF EXISTS update_schema_embeddings_updated_at ON schema_embeddings;
CREATE TRIGGER update_schema_embeddings_updated_at
    BEFORE UPDATE ON schema_embeddings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_kpi_definitions_updated_at ON kpi_definitions;
CREATE TRIGGER update_kpi_definitions_updated_at
    BEFORE UPDATE ON kpi_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_business_glossaries_updated_at ON business_glossaries;
CREATE TRIGGER update_business_glossaries_updated_at
    BEFORE UPDATE ON business_glossaries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create a function to calculate cosine similarity (for reference)
CREATE OR REPLACE FUNCTION cosine_similarity(a vector, b vector)
RETURNS float AS $cosine$
BEGIN
    RETURN 1 - (a <=> b);
END;
$cosine$ LANGUAGE plpgsql IMMUTABLE STRICT;

-- Create a function to search similar embeddings
CREATE OR REPLACE FUNCTION search_similar_embeddings(
    query_embedding vector,
    data_source_filter INTEGER DEFAULT NULL,
    element_type_filter TEXT DEFAULT NULL,
    similarity_threshold FLOAT DEFAULT 0.7,
    result_limit INTEGER DEFAULT 10
)
RETURNS TABLE (
    id INTEGER,
    data_source_id INTEGER,
    schema_id INTEGER,
    element_type TEXT,
    element_name TEXT,
    content TEXT,
    metadata JSONB,
    similarity_score FLOAT
) AS $search$
BEGIN
    RETURN QUERY
    SELECT 
        se.id,
        se.data_source_id,
        se.schema_id,
        se.element_type,
        se.element_name,
        se.content,
        se.metadata,
        cosine_similarity(se.embedding, query_embedding) as similarity_score
    FROM schema_embeddings se
    WHERE 
        se.deleted_at IS NULL
        AND (data_source_filter IS NULL OR se.data_source_id = data_source_filter OR se.data_source_id = 0)
        AND (element_type_filter IS NULL OR se.element_type = element_type_filter)
        AND cosine_similarity(se.embedding, query_embedding) >= similarity_threshold
    ORDER BY similarity_score DESC
    LIMIT result_limit;
END;
$search$ LANGUAGE plpgsql;

-- Add comments to tables
COMMENT ON TABLE schema_embeddings IS 'Stores vector embeddings for schema elements (tables, columns, KPIs, glossary terms)';
COMMENT ON TABLE kpi_definitions IS 'Stores business KPI definitions and formulas';
COMMENT ON TABLE business_glossaries IS 'Stores business term definitions and glossary';
COMMENT ON TABLE rag_query_contexts IS 'Stores context information for NL2SQL queries';

-- Add comments to important columns
COMMENT ON COLUMN schema_embeddings.embedding IS 'Vector embedding (1536 dimensions for OpenAI ada-002)';
COMMENT ON COLUMN schema_embeddings.element_type IS 'Type of element: table, column, kpi, glossary';
COMMENT ON COLUMN schema_embeddings.metadata IS 'Additional metadata in JSON format';
COMMENT ON COLUMN kpi_definitions.formula IS 'SQL formula or calculation for the KPI';
COMMENT ON COLUMN business_glossaries.synonyms IS 'Alternative terms in JSON array format';
COMMENT ON COLUMN rag_query_contexts.context IS 'Retrieved context from RAG system in JSON format';

-- +goose Down
-- Drop all RAG tables and functions
DROP TABLE IF EXISTS rag_query_contexts;
DROP TABLE IF EXISTS business_glossaries;
DROP TABLE IF EXISTS kpi_definitions;
DROP TABLE IF EXISTS schema_embeddings;
DROP FUNCTION IF EXISTS search_similar_embeddings(vector, INTEGER, TEXT, FLOAT, INTEGER);
DROP FUNCTION IF EXISTS cosine_similarity(vector, vector);
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP EXTENSION IF EXISTS vector;