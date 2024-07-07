package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"

	"github.com/smakimka/balb/internal/model"
	"github.com/smakimka/balb/internal/server/storage"
)

type AddUserHandler struct {
	s storage.Storage
}

func NewAdduserHandler(s storage.Storage) AddUserHandler {
	return AddUserHandler{s: s}
}

func (h AddUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := &model.User{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, model.Response{Msg: "wrong json"})
		return
	}

	_, err := h.s.CreateUser(r.Context(), data)
	if err != nil {
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, model.Response{Msg: "user already exists"})
			return
		}

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, model.Response{Msg: "internal server error"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, model.Response{})
}
