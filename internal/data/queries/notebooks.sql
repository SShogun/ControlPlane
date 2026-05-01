-- name: GetNotebookByID :one
SELECT
	id,
	title,
	content,
	is_published,
	created_at,
	updated_at
FROM notebooks
WHERE id = $1;

-- name: ListPublishedNotebooks :many
SELECT
	id,
	title,
	content,
	is_published,
	created_at,
	updated_at
FROM notebooks
WHERE is_published = true
ORDER BY created_at DESC;

-- name: UpdateNotebookDraft :exec
UPDATE notebooks
SET
	title = $1,
	content = $2,
	updated_at = CURRENT_TIMESTAMP
WHERE id = $3
	AND is_published = false;

-- name: SearchNotebooks :many
SELECT
	id,
	title,
	content,
	is_published,
	created_at,
	updated_at
FROM notebooks
WHERE title ILIKE '%' || $1 || '%'
	 OR content ILIKE '%' || $2 || '%'
ORDER BY created_at DESC;

