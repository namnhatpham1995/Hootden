package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
)

const minPasswordLength = 8

// dummyHash is compared against on a login attempt for an email that has no
// account, or an account with no password set, so a bcrypt comparison always
// runs and the response takes the same time either way.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("hootden-dummy-hash"), bcrypt.DefaultCost)

type PasswordHandlers struct {
	Pool         *pgxpool.Pool
	Store        PasswordStore
	CookieDomain string
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a new account with a password, then signs it in exactly
// as Callback does for a Google account.
func (h PasswordHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Password) < minPasswordLength {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	userID, err := h.Store.CreateUserWithPassword(r.Context(), normalizeEmail(req.Email), string(hash))
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			httpapi.WriteJSONError(w, http.StatusConflict, "email already in use")
			return
		}
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	rawToken, err := CreateSession(r.Context(), h.Pool, userID)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	setSessionCookie(w, h.CookieDomain, rawToken, SessionTTL)
	w.WriteHeader(http.StatusNoContent)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login verifies email/password and signs in. An unknown email, an account
// with no password set (Google-only), and a wrong password all take the
// same bcrypt-comparison path and return the same response, so none of the
// three is distinguishable from the outside.
func (h PasswordHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, passwordHash, err := h.Store.FindUserByEmail(r.Context(), normalizeEmail(req.Email))
	if err != nil && !errors.Is(err, ErrAccountNotFound) {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "sign-in failed")
		return
	}

	found := err == nil && passwordHash != ""
	hash := dummyHash
	if found {
		hash = []byte(passwordHash)
	}
	match := bcrypt.CompareHashAndPassword(hash, []byte(req.Password)) == nil

	if !found || !match {
		httpapi.WriteJSONError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	rawToken, err := CreateSession(r.Context(), h.Pool, userID)
	if err != nil {
		httpapi.WriteJSONError(w, http.StatusInternalServerError, "sign-in failed")
		return
	}

	setSessionCookie(w, h.CookieDomain, rawToken, SessionTTL)
	w.WriteHeader(http.StatusNoContent)
}
