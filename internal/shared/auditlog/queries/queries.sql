-- name: InsertAuditLog :exec
INSERT INTO audit_logs (id, request_id, user_id, user_email, method, path, status_code, duration_ms, ip, user_agent, request_body, response_size, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: InsertErrorLog :exec
INSERT INTO error_logs (id, request_id, user_id, user_email, level, message, error, stack_trace, method, path, status_code, ip, user_agent, request_body, metadata, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16);
