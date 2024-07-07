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

type GetUserHandler struct {
	s storage.Storage
}

func NewGetUserHandler(s storage.Storage) GetUserHandler {
	return GetUserHandler{s: s}
}

func (h GetUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userUID := chi.URLParam(r, "userUID")
	front := chi.URLParam(r, "front")

	if userUID == "" || front == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, model.Response{Msg: "wrong data"})
		return
	}

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

	user, err := h.s.GetUser(r.Context(), frontInt, userUID)
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
	render.JSON(w, r, user)
}
