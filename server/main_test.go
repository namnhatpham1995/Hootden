package main

import (
	"net/http"
	"testing"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
	"github.com/namnhatpham1995/Hootden/server/internal/page"
)

func TestGetRoutesAreReadOnly(t *testing.T) {
	routes := buildRoutes(nil, auth.Handlers{}, auth.PasswordHandlers{}, page.Handlers{}, false, "")
	if err := checkGetRoutesAreReadOnly(routes); err != nil {
		t.Error(err)
	}
}

func TestGetRoutesAreReadOnly_CatchesStateChangingGET(t *testing.T) {
	routes := []route{
		{Method: http.MethodGet, Path: "/pages/{id}/delete", Handler: http.NotFoundHandler()},
	}
	if err := checkGetRoutesAreReadOnly(routes); err == nil {
		t.Fatal("expected a GET route not marked ReadOnly and not the named exception to be rejected")
	}
}
