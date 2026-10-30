package notebookauth

import (
	"errors"
	"testing"
)

func TestDraftAuthorization(t *testing.T) {
	tests := []struct {
		name                string
		actor, author, team int
		member              bool
		status              string
		want                error
	}{
		{"owner without team", 7, 7, 0, false, "draft", nil},
		{"owner with membership", 7, 7, 5, true, "draft", nil},
		{"other user", 8, 7, 0, false, "draft", ErrForbidden},
		{"team member not owner", 8, 7, 5, true, "draft", ErrForbidden},
		{"owner no longer member", 7, 7, 5, false, "draft", ErrForbidden},
		{"zero actor", 0, 0, 0, false, "draft", ErrForbidden},
		{"owner non-draft", 7, 7, 0, false, "submitted", ErrNotDraft},
		{"unauthorized non-draft", 8, 7, 0, false, "submitted", ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AuthorizeDraft(tt.actor, tt.author, tt.team, tt.member, tt.status)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v; want %v", err, tt.want)
			}
		})
	}
}
