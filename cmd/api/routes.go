package main

//we are using chi - router middleware to handle secure routes
//go get -u github.com/go-chi/chi/v5

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

//routes generates our routes and attaches them to handlers, using the chi router
//note that we return type http.Handler, and not *chi.Mux; since chi.Mux satisfies
// the interface requirements for http.Handler, it makes sense to return the type
// that is part of the standard library

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)

	//In order for our vue-client to access our api we need to enable it with from our CORS
	//Lets get the chi CORS package by running go get github.com/go-chi/cors
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"}, //change in production to front-end url
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Post("/users/login", app.Login)
	mux.Post("/users/logout", app.Logout)
	mux.Post("/validate-token", app.ValidateToken)
	//mux.Post("/blogs", app.AllBlogs)

	mux.Get("/blogs", app.AllBlogs)
	mux.Get("/blogs/{slug}", app.OneBlog)

	// protected routes
	// use AuthTokenMiddleware meaning all the users need to have a token to be able to access them
	// all the routes inside the block are prefix with /admin
	mux.Route("/admin", func(mux chi.Router) {
		// This is for protecting our route but it doesnt allow vue have acess to it so uncomment later when u find a solution
		//mux.Use(app.AuthTokenMiddleware)

		mux.Post("/users", app.AllUsers)
		mux.Post("/users/save", app.EditUser)
		mux.Post("/users/get/{id}", app.GetUser)
		mux.Post("/users/delete", app.DeleteUser)
		mux.Post("/log-user-out/{id}", app.LogUserOutAndSetInactive)

		//admin blog routes
		mux.Post("/blogs/save", app.EditBlog)
		mux.Post("/blogs/{id}", app.BlogByID)
		mux.Post("/blogs/delete", app.DeleteBlog)

	})

	// static files
	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return mux
}
