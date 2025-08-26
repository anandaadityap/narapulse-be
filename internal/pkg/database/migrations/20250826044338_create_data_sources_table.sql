-- +goose Up
-- +goose StatementBegin

-- Create data_sources table
CREATE TABLE data_sources (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'inactive',
    config JSONB,
    metadata JSONB,
    last_tested TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_data_sources_user_id ON data_sources(user_id);
CREATE INDEX idx_data_sources_deleted_at ON data_sources(deleted_at);

-- Create schemas table
CREATE TABLE schemas (
    id SERIAL PRIMARY KEY,
    data_source_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    columns JSONB,
    row_count BIGINT DEFAULT 0,
    sample_data JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    FOREIGN KEY (data_source_id) REFERENCES data_sources(id) ON DELETE CASCADE
);

-- Create indexes for schemas table
CREATE INDEX idx_schemas_data_source_id ON schemas(data_source_id);
CREATE INDEX idx_schemas_deleted_at ON schemas(deleted_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop tables in reverse order
DROP TABLE IF EXISTS schemas;
DROP TABLE IF EXISTS data_sources;

-- +goose StatementEnd
