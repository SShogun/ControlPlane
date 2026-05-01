-- this is like the blackbox for the entire system where you ONLY INSERT not UPDATE or DELETE 
-- name: InsertAuditLog :exec
INSERT INTO audit_events (actor_id, event_type, entity_type, entity_id)
VALUES ($1, $2, $3, $4);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 100;