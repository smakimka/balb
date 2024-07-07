package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/smakimka/balb/internal/bot/handlers"
	"github.com/smakimka/balb/internal/bot/storage"
)

func New(s storage.Storage) chi.Router {
	notifyHandler := handlers.NewNotifyHandler(s)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/notify", notifyHandler.ServeHTTP)

	return r
}
