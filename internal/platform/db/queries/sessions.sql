-- name: CreateSession :one
INSERT INTO sessions (
    id,
    user_id,
    token_hash,
    ip_address,
    user_agent,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, user_id, ip_address, user_agent, expires_at, revoked_at, created_at;

-- name: GetActiveSessionByHash :one
SELECT id, user_id, ip_address, user_agent, expires_at, revoked_at, created_at
FROM sessions
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > NOW()
LIMIT 1;

-- name: RevokeSessionByID :exec
UPDATE sessions
SET revoked_at = NOW()
WHERE id = $1
  AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE sessions
SET revoked_at = NOW()
WHERE user_id = $1
  AND revoked_at IS NULL;
