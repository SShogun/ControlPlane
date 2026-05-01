-- Get a single notebook by ID (viewNotebook)
SELECT
	id,
	title,
	content,
	is_published,
	created_at,
	updated_at
FROM notebooks
WHERE id = ?;

-- List all published notebooks (listNotebooks)
SELECT
	id,
	title,
	content,
	is_published,
	created_at,
	updated_at
FROM notebooks
WHERE is_published = 1
ORDER BY created_at DESC;

-- Update an existing draft (editDraftForm)
UPDATE notebooks
SET
	title = ?,
	content = ?,
	updated_at = CURRENT_TIMESTAMP
WHERE id = ?
	AND is_published = 0;

-- Search notebooks by title or content (searchNotebooks)
SELECT
	id,
	title,
	content,
	is_published,
	created_at,
	updated_at
FROM notebooks
WHERE title LIKE '%' || ? || '%'
	 OR content LIKE '%' || ? || '%'
ORDER BY created_at DESC;
