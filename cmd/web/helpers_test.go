package web

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadIDParam(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "valid id", value: "123", want: 123},
		{name: "non numeric id", value: "abc", want: 0},
		{name: "missing id", value: "", want: 0},
	}

	app := newTestApplication(nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = addRouteParam(req, "id", tt.value)

			if got := app.readIDParam(req, "id"); got != tt.want {
				t.Fatalf("expected id %d; got %d", tt.want, got)
			}
		})
	}
}

func TestRenderUsesTemplateCache(t *testing.T) {
	app := newTestApplication(nil, map[string]*template.Template{
		"page.tmpl": parseTemplate("page.tmpl", "hello {{.Flash}}"),
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	app.render(rr, req, http.StatusAccepted, "page.tmpl", &templateData{Flash: "world"})

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d; got %d", http.StatusAccepted, rr.Code)
	}
	if got := rr.Body.String(); got != "hello world" {
		t.Fatalf("expected rendered body; got %q", got)
	}
}
