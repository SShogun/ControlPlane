-- List teams
SELECT id, name, created_at, updated_at
FROM teams
ORDER BY name;

-- Get team by id
SELECT id, name, created_at, updated_at
FROM teams
WHERE id = $1;

-- List team members
SELECT u.id, u.email, u.password_hash
FROM users u
JOIN memberships m ON m.user_id = u.id
WHERE m.team_id = $1;

-- Add membership
INSERT INTO memberships (user_id, team_id, role)
VALUES ($1, $2, $3);

-- Create notebook revision
INSERT INTO notebook_revisions (notebook_id, author_id, title, body, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- Get latest revision for notebook
SELECT id, notebook_id, author_id, title, body, status, created_at, updated_at
FROM notebook_revisions
WHERE notebook_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- Create tag
INSERT INTO tags (name)
VALUES ($1)
RETURNING id;

-- Attach tag to notebook
INSERT INTO notebook_tags (notebook_id, tag_id)
VALUES ($1, $2);

-- List notebook tags
SELECT t.id, t.name, t.created_at
FROM tags t
JOIN notebook_tags nt ON nt.tag_id = t.id
WHERE nt.notebook_id = $1;