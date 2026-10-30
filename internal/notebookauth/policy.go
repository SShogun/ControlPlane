package notebookauth

import "errors"

var (
	ErrNotFound  = errors.New("notebook not found")
	ErrForbidden = errors.New("notebook modification forbidden")
	ErrNotDraft  = errors.New("latest revision is not a draft")
)

func AuthorizeDraft(actorID, authorID, teamID int, member bool, status string) error {
	if actorID <= 0 || actorID != authorID || (teamID != 0 && !member) {
		return ErrForbidden
	}
	if status != "draft" {
		return ErrNotDraft
	}
	return nil
}
