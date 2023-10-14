package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"thelsblog-server/internal/data"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mozillazg/go-slugify"
)

var staticPath = "./static/"

type jsonResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type envelope map[string]interface{}

func (app *application) Login(w http.ResponseWriter, r *http.Request) {
	type credentials struct {
		UserName string `json:"email"`
		Password string `json:"password"`
	}

	var creds credentials

	var payload jsonResponse

	//We created a read and write json function in helpers.go so we dont ave to manually write the code everytime
	err := app.readJSON(w, r, &creds)
	if err != nil {
		app.errorLog.Println(err)
		payload.Error = true
		payload.Message = "invalid json supplied or json missing entirely"
		_ = app.writeJSON(w, http.StatusBadRequest, payload)
	}

	//AUTHENTICATE
	app.infoLog.Println(creds.UserName, creds.Password)

	// look up user by email
	user, err := app.models.User.GetByEmail(creds.UserName)
	if err != nil {
		app.errorJSON(w, errors.New("invalid username/password"))
		return
	}

	// validate the users  password
	validPassword, err := user.PasswordMatches(creds.Password)
	if err != nil || !validPassword {
		app.errorJSON(w, errors.New("invalid username/password"))
		return
	}
	// make sure user is active
	if user.Active == 0 {
		app.errorJSON(w, errors.New("user is not active"))
		return
	}

	// we have a valid user, so generate a token
	token, err := app.models.Token.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// save it to database
	err = app.models.Token.Insert(*token, *user)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// send back a response
	payload = jsonResponse{
		Error:   false,
		Message: "logged in",

		// data envelope sent to front end to be able to store user amd token as cookie
		Data: envelope{"token": token, "user": user},
	}

	//We use our write json func we created at helper.go
	err = app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		app.errorLog.Println(err)
	}

}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Token string `json:"token"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, errors.New("invalid json"))
		return
	}

	err = app.models.Token.DeleteByToken(requestPayload.Token)
	if err != nil {
		app.errorJSON(w, errors.New("invalid json"))
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "logged out",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	var users data.User
	all, err := users.GetAll()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "success",
		Data:    envelope{"users": all},
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// Edit user handler
func (app *application) EditUser(w http.ResponseWriter, r *http.Request) {
	var user data.User
	err := app.readJSON(w, r, &user)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if user.ID == 0 {
		//add user
		if _, err := app.models.User.Insert(user); err != nil {
			app.errorJSON(w, err)
			return
		}
	} else {
		// editing user
		u, err := app.models.User.GetByID(user.ID)
		if err != nil {
			app.errorJSON(w, err)
			return
		}
		u.Email = user.Email
		u.FirstName = user.FirstName
		u.LastName = user.LastName
		u.Active = user.Active

		if err := u.Update(); err != nil {
			app.errorJSON(w, err)
			return
		}

		// if password != string, update password
		if user.Password != "" {
			err := u.ResetPassword(user.Password)
			if err != nil {
				app.errorJSON(w, err)
				return
			}
		}
	}
	payload := jsonResponse{
		Error:   false,
		Message: "Changes saved",
	}

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

// Get user handler
func (app *application) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	user, err := app.models.User.GetByID(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, user)
}

func (app *application) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	err = app.models.User.DeleteByID(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "User deleted",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

// set a user as inactive when hey are log out
func (app *application) LogUserOutAndSetInactive(w http.ResponseWriter, r *http.Request) {
	//gets userID
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	user, err := app.models.User.GetByID(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Sets user to inactive and saves to the database
	user.Active = 0
	err = user.Update()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	//delete token for user
	err = app.models.Token.DeleteTokensForUser(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "user logged out and set to inactive",
	}

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *application) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Token string `json:"token"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	valid := false
	valid, _ = app.models.Token.ValidToken(requestPayload.Token)

	payload := jsonResponse{
		Error: false,
		Data:  valid,
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

// Display a list of all our blogs
func (app *application) AllBlogs(w http.ResponseWriter, r *http.Request) {
	blogs, err := app.models.Blog.GetAll()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "success",
		Data:    envelope{"blogs": blogs},
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// Get only one blog based on their slug
func (app *application) OneBlog(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	blog, err := app.models.Blog.GetOneBySlug(slug)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error: false,
		Data:  blog,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// get all creators
func (app *application) EditBlog(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		CreatedByID  int    `json:"createdby_id"`
		Description  string `json:"description"`
		Content      string `json:"content"`
		BannerBase64 string `json:"banner"`
		CategoryIDs  []int  `json:"category_ids"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	blog := data.Blog{
		ID:          requestPayload.ID,
		Title:       requestPayload.Title,
		CreatedByID: requestPayload.CreatedByID,
		Description: requestPayload.Description,
		Content:     requestPayload.Content,
		Slug:        slugify.Slugify(requestPayload.Title),
		CategoryIDs: requestPayload.CategoryIDs,
	}

	// image decoding to check if we have a banner
	if len(requestPayload.BannerBase64) > 0 {
		// we have a banner
		decoded, err := base64.StdEncoding.DecodeString(requestPayload.BannerBase64)
		if err != nil {
			app.errorJSON(w, err)
			return
		}

		// write image to /static/banners
		if err := os.WriteFile(fmt.Sprintf("%s/banners/%s.jpg", staticPath, blog.Slug), decoded, 0666); err != nil {
			app.errorJSON(w, err)
			return
		}

	}

	if blog.ID == 0 {
		// adding a blog
		_, err := app.models.Blog.Create(blog)
		if err != nil {
			app.errorJSON(w, err)
			return
		}
	} else {
		// update a blog
		err := blog.Update()
		if err != nil {
			app.errorJSON(w, err)
			return
		}
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Changes saved",
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

// get blog by id
func (app *application) BlogByID(w http.ResponseWriter, r *http.Request) {
	blogID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	blog, err := app.models.Blog.GetOneById(blogID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error: false,
		Data:  blog,
	}
	app.writeJSON(w, http.StatusOK, payload)

}

// delete blog
func (app *application) DeleteBlog(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload.ID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	err = app.models.Blog.DeleteByID(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Blog Deleted",
	}

	app.writeJSON(w, http.StatusOK, payload)
}
