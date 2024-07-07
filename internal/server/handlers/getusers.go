package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/smakimka/balb/internal/model"
	"github.com/smakimka/balb/internal/server/storage"
)

type GetUsersHandler struct {
	s storage.Storage
}

func NewGetUsersHandler(s storage.Storage) GetUsersHandler {
	return GetUsersHandler{s: s}
}

func (h GetUsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	front := chi.URLParam(r, "front")

	frontInt, err := strconv.Atoi(front)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, model.Response{Msg: "wrong data"})
		return
	}

	if frontInt < model.TelegramFront || frontInt > model.TelegramFront {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, model.Response{Msg: "wrong data"})
		return
	}

	users, err := h.s.GetUsers(r.Context(), frontInt)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, model.Response{Msg: "user not found"})
			return
		}

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, model.Response{Msg: "internal server error"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, users)
}
