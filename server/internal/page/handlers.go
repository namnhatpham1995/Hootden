package page

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
	"github.com/namnhatpham1995/Hootden/server/internal/workspace"
)

type Handlers struct {
	Pool *pgxpool.Pool
}

// writeOwnershipError maps workspace.ErrNotFound to 404 and anything else
// to 500, uniformly, so a page or parent that doesn't exist and one that
// belongs to someone else are indistinguishable to the caller.
func writeOwnershipError(w http.ResponseWriter, err error) {
	if errors.Is(err, workspace.ErrNotFound) {
		httpapi.WriteJSONError(w, http.StatusNotFound, "not found")
		return
	}
	httpapi.WriteJSONError(w, http.StatusInternalServerError, "request failed")
}

// List returns the caller's Den as a flat, doc-free list.
func (h Handlers) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	denID, err := workspace.DenIDForUser(r.Context(), h.Pool, userID)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to load pages")
		return
	}

	nodes, err := Tree(r.Context(), h.Pool, denID)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to load pages")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

// Get returns a single page including its document body.
func (h Handlers) Get(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	pageID := r.PathValue("id")

	if err := workspace.RequireOwnPage(r.Context(), h.Pool, userID, pageID); err != nil {
		writeOwnershipError(w, err)
		return
	}

	p, err := Get(r.Context(), h.Pool, pageID)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to load page")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

type createRequest struct {
	ParentID string `json:"parent_id"`
	Title    string `json:"title"`
}

func (h Handlers) Create(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	denID, err := workspace.DenIDForUser(r.Context(), h.Pool, userID)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to create page")
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ParentID != "" {
		if err := workspace.RequireOwnPage(r.Context(), h.Pool, userID, req.ParentID); err != nil {
			writeOwnershipError(w, err)
			return
		}
	}

	n, err := Create(r.Context(), h.Pool, denID, req.ParentID, req.Title)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to create page")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(n)
}

type updateRequest struct {
	Title    *string         `json:"title,omitempty"`
	ParentID *string         `json:"parent_id,omitempty"`
	Position *int            `json:"position,omitempty"`
	Doc      json.RawMessage `json:"doc,omitempty"`
}

// Update renames and/or moves a page, either independently or together in
// one request.
func (h Handlers) Update(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	pageID := r.PathValue("id")

	if err := workspace.RequireOwnPage(r.Context(), h.Pool, userID, pageID); err != nil {
		writeOwnershipError(w, err)
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title != nil {
		if err := Rename(r.Context(), h.Pool, pageID, *req.Title); err != nil {
			httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to rename page")
			return
		}
	}

	if req.Doc != nil {
		if err := SaveDoc(r.Context(), h.Pool, pageID, req.Doc); err != nil {
			httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to save document")
			return
		}
	}

	if req.Position != nil {
		newParentID := ""
		if req.ParentID != nil {
			newParentID = *req.ParentID
		} else {
			// No parent_id given: keep the current one.
			current, err := currentParentID(r.Context(), h.Pool, pageID)
			if err != nil {
				httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to move page")
				return
			}
			newParentID = current
		}

		if newParentID != "" {
			if err := workspace.RequireOwnPage(r.Context(), h.Pool, userID, newParentID); err != nil {
				writeOwnershipError(w, err)
				return
			}
		}

		if err := Move(r.Context(), h.Pool, pageID, newParentID, *req.Position); err != nil {
			if errors.Is(err, ErrCycle) {
				httpapi.WriteJSONError(w, http.StatusBadRequest, "cannot move a page under its own descendant")
				return
			}
			httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to move page")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	pageID := r.PathValue("id")

	if err := workspace.RequireOwnPage(r.Context(), h.Pool, userID, pageID); err != nil {
		writeOwnershipError(w, err)
		return
	}

	if err := Delete(r.Context(), h.Pool, pageID); err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to delete page")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
