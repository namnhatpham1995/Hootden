package workspace

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
)

type meResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Workspace workspace `json:"workspace"`
}

type workspace struct {
	ID string `json:"id"`
}

// Me resolves entirely from the session -- no id is ever accepted as input,
// so there's nothing for an ownership check to guard here.
func Me(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())

		var resp meResponse
		resp.ID = userID
		err := pool.QueryRow(r.Context(), `
			SELECT u.email, w.id
			FROM users u
			JOIN workspaces w ON w.owner_id = u.id AND w.personal = true
			WHERE u.id = $1
		`, userID).Scan(&resp.Email, &resp.Workspace.ID)
		if err != nil {
			httpapi.WriteJSONError(w, http.StatusInternalServerError, "failed to load account")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
