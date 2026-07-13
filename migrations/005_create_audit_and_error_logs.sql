-- +goose Up

CREATE TABLE IF NOT EXISTS  audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id VARCHAR(100) NOT NULL,
    user_id UUID,
    user_email VARCHAR(255),
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms BIGINT NOT NULL,
    ip VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    request_body TEXT,
    response_size INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS  error_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id VARCHAR(100) NOT NULL,
    user_id UUID,
    user_email VARCHAR(255),
    level VARCHAR(20) NOT NULL DEFAULT 'error',
    message TEXT NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    stack_trace TEXT NOT NULL DEFAULT '',
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    status_code INTEGER NOT NULL,
    ip VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    request_body TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_method_path ON audit_logs(method, path);

CREATE INDEX IF NOT EXISTS idx_error_logs_request_id ON error_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_error_logs_level ON error_logs(level);
CREATE INDEX IF NOT EXISTS idx_error_logs_created_at ON error_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_error_logs_status_code ON error_logs(status_code);

-- +goose Down

DROP TABLE IF EXISTS error_logs;
DROP TABLE IF EXISTS audit_logs;
