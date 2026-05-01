-- name: ApproveRevision :exec
UPDATE notebook_revisions
SET status = 'approved'
WHERE id = $1;

-- name: UpdatePublishedRevision :exec
UPDATE notebooks
SET current_published_revision_id = $1
WHERE id = $2;