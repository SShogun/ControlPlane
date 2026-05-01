-- a queue system which exists in states: draft -> submitted -> approved (or rejected)

-- List drafts waiting for review
SELECT id, document_id, author_id, title, status, created_at 
FROM document_revisions 
WHERE status = 'submitted'
ORDER BY created_at ASC;

-- Update a revision's status to approved or rejected
UPDATE document_revisions 
SET status = $1 
WHERE id = $2;

-- Update a parent document's published_revision_id
UPDATE documents 
SET published_revision_id = $1 
WHERE id = $2;

