package handlers

import (
	"errors"
	"github.com/go-chi/render"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"net/http"
)

type Authorizer interface {
	CheckPassword(username, passwordHash string) (int, error)
}

func Authorize(w http.ResponseWriter, r *http.Request, a Authorizer) (int, string, string) {
	userId := int(r.Context().Value("user_id").(float64))
	username := r.Context().Value("username").(string)
	passwordHash := r.Context().Value("password_hash").(string)

	id, err := a.CheckPassword(username, passwordHash)
	if err != nil {
		if errors.Is(err, storage.ErrPasswordUnmatched) {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, errresp.Error("invalid token claims: password"))

			return 0, "", ""
		}
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errresp.Error("failed to check user password"))

		return 0, "", ""
	}

	if id != userId {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, errresp.Error("invalid token claims: id"))

		return 0, "", ""
	}

	return id, username, passwordHash
}
