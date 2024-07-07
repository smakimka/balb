package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/smakimka/balb/internal/server/handlers"
	"github.com/smakimka/balb/internal/server/storage"
)

func New(s storage.Storage) chi.Router {
	getUserHandler := handlers.NewGetUserHandler(s)
	getUsersHandler := handlers.NewGetUsersHandler(s)
	addUserHandler := handlers.NewAdduserHandler(s)
	subscribeHander := handlers.NewSubscribeHandler(s)
	unsubscribeHandler := handlers.NewUnsubscribeHandler(s)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/users", func(r chi.Router) {
		r.Post("/add", addUserHandler.ServeHTTP)
		r.Get("/get/{front}", getUsersHandler.ServeHTTP)
		r.Get("/get/{front}/{userUID}", getUserHandler.ServeHTTP)
	})

	r.Route("/subscriptions", func(r chi.Router) {
		r.Post("/subscribe", subscribeHander.ServeHTTP)
		r.Post("/unsubscribe", unsubscribeHandler.ServeHTTP)
	})

	return r
}
