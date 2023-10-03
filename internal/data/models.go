package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// database timeout
const dbTimeout = time.Second * 3

var db *sql.DB

func New(dbPool *sql.DB) Models {
	db = dbPool

	return Models{
		User:  User{},
		Token: Token{},
	}
}

type Models struct {
	User  User
	Token Token
}

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Token     Token     `json:"token"`
}

// Query to get all users
// we are returning a slice of all of the user, sorted by last name
func (u *User) GetAll() ([]*User, error) {
	// Important to shut down db once Query operation is carried out
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// SQL query statement that will be passed as a parameter to our db Query context
	query := `select id, email, first_name, last_name, password, created_at, updated_at  from users order by last_name`

	// Query every row for users and close when  done
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// varibale user points at an slice of User. we will be returning every field in that struct
	var users []*User

	// for loop to each individual field in our User slice one by one so they can be individually scanned for errors before we return any of them
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Password,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// if no errors are found each field will be appended back to the slice users
		users = append(users, &user)
	}
	return users, nil

}

// Query a user by their email
// since email is unique we will only get one row of user
func (u *User) GetByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	//SQL query
	query := `select id, email, first_name, last_name, password, created_at, updated_at from users where email = $1`

	// variable for User type the function will return
	var user User

	// Query the database for only ONE row,
	// QueryRowContext takes in 3 parameters, ctx, query and  what we want to query by, email, name, id etc
	row := db.QueryRowContext(ctx, query, email)

	// scan for errors
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	// if error
	if err != nil {
		return nil, err
	}

	//else no error return refernce to user we pointed at, and nil(no error)
	return &user, nil
}

// Get (one) user by their ID
func (u *User) GetByID(id int) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, email, first_name, last_name, password, created_at, updated_at from users where id = $1`

	// variable for User type the function will return
	var user User

	//Query the database for only ONE row, QueryRowContext takes in 3 parameters, ctx(context.Context), query and  what we want to query by, email, name, id etc
	row := db.QueryRowContext(ctx, query, id)

	// scan for errors on each individual field
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	// if error
	if err != nil {
		return nil, err
	}

	//else if no error then return refernce to user we pointed at, and nil(no error)
	return &user, nil
}

// Update one User in the database
func (u *User) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// next sql statement, this time not a query to return anything but a statement bc we are updating
	// You can also namae the variable query or leave as stmt(statement)
	// Remember in SQL dont put a comma (,) before the where id
	stmt := `update users set
			email = $1,
			first_name = $2,
			last_name = $3,
			updated_at =$4
			where id = $5
			`

	//ExecContext doesnt return anything, we just want to query for errors and nit return anything
	_, err := db.ExecContext(ctx, stmt,
		u.Email,
		u.FirstName,
		u.LastName,
		time.Now(),
		u.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

// Delete user by ID
func (u *User) Delete() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `delete from users where id = $1`

	_, err := db.ExecContext(ctx, stmt, u.ID)

	if err != nil {
		return err
	}

	return nil

}

// Insert New user into the database and return their ID
func (u *User) Insert(user User) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// This generates hashed password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
	if err != nil {
		return 0, err
	}

	// variable to store the new user ID
	var newID int

	stmt := `insert into users (email, first_name, last_name, password, created_at, updated_at)
			values ($1, $2, $3, $4 , $5, $6) returning id
			`

	err = db.QueryRowContext(ctx, stmt,
		user.Email,
		user.FirstName,
		user.LastName,
		hashedPassword,
		time.Now(),
		time.Now(),
	).Scan(&newID)

	if err != nil {
		return 0, err
	}

	return newID, nil

}

// Reset Password
func (u *User) ResetPassword(password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	stmt := `update users set password = $1 where id = $2`

	_, err = db.ExecContext(ctx, stmt, hashedPassword, u.ID)
	if err != nil {
		return err
	}

	return nil
}

// Using Go's bcrypt package to compare user supplied passowrd
// Check if password user is providing when hashed matches the hashed password in our database
func (u *User) PasswordMatches(plainText string) (bool, error) {
	//compare passowrds using bcrypt package
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainText))

	if err != nil {
		// We can get two kinds of error in this situation, error when password doesnt match
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			// invalid password
			return false, nil

		//and error when something unexpected happens
		default:
			return false, err
		}
	}

	//else password matches
	return true, nil
}

// User token struct
// Token is the struct for any token in the database
// Any json fied with - means that field will not be exported to JSON,
// we dont want to export our TokenHashed to json
type Token struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	TokenHash []byte    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Expiry    time.Time `json:"expiry"`
}

// Get a User by their Token
func (t *Token) GetByToken(plainText string) (*Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, user_id, email, token, token_hash, created_at, updated_at, expiry
			from tokens where token = $1
			`

	var token Token
	row := db.QueryRowContext(ctx, query, plainText)
	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.Email,
		&token.Token,
		&token.TokenHash,
		&token.CreatedAt,
		&token.UpdatedAt,
		&token.Expiry,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (t *Token) GetUserForToken(plainText string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, email, first_name, last_name, password, created_at, updated_at from users where email = $1`

	// variable for User type the function will return
	var user User

	//Query the database for only ONE row, QueryRowContext takes in 3 parameters, ctx(context.Context), query and  what we want to query by, email, name, id etc
	row := db.QueryRowContext(ctx, query, t.UserID)

	// scan for errors on each individual field
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	// if error
	if err != nil {
		return nil, err
	}

	//else if no error then return user we pointed at, and nil(no error)
	return &user, nil
}

// Generate Token
func (t *Token) GenerateToken(userID int, ttl time.Duration) (*Token, error) {
	// we only need these two fields from our token struct to generate a user toekn
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Token = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Token))
	token.TokenHash = hash[:]

	return token, nil
}

// Authenticate token
// AUthentication takes the full http request, extracts the authorization header,
// takes the plain text token from that header and looks up the associated token entry
// in the database, and then finds the user associated with that token. If the token
// is valid and a user is found, the user is returned; otherwise an error is returned
func (t *Token) AuthenticateToken(r *http.Request) (*User, error) {
	// Get the authorization header from the frontend
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("no authorization header received")
	}

	// Get the plain text token from the header
	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no valid authorization header received")
	}

	token := headerParts[1]

	//make sure the token is of the correct length
	if len(token) != 26 {
		return nil, errors.New("token wrong size")
	}

	// Get the token from the database, using the plaintext token to find it
	tokn, err := t.GetByToken(token)
	if err != nil {
		return nil, errors.New("expired token")
	}

	// ensure the tokens are not expire
	if tokn.Expiry.Before(time.Now()) {
		return nil, errors.New("expired token")
	}

	// get the user associated with the token
	user, err := t.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("no matching user found")
	}
	return user, nil
}

// insert token
func (t *Token) Insert(token Token, u User) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// delete any existing tokens
	stmt := `delete from tokens where user_id = $1`
	_, err := db.ExecContext(ctx, stmt, token.UserID)
	if err != nil {
		return err
	}

	token.Email = u.Email

	stmt = `insert into tokens (user_id, email, token, token_hash, created_at,  updated_at, expiry)
		values ($1, $2, $3, $4 , $5, $6, $7)`

	_, err = db.ExecContext(ctx, stmt,
		token.UserID,
		token.Email,
		token.Token,
		token.TokenHash,
		time.Now(),
		time.Now(),
		token.Expiry,
	)
	if err != nil {
		return err
	}

	return nil
}

// Delete a token
func (t *Token) DeleteByToken(plainText string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `delete from tokens where token = $1`

	_, err := db.ExecContext(ctx, stmt, plainText)
	if err != nil {
		return err
	}

	return nil
}

func (t *Token) ValidToken(plainText string) (bool, error) {
	token, err := t.GetByToken(plainText)
	if err != nil {
		return false, errors.New("no matching user found")
	}

	if token.Expiry.Before(time.Now()) {
		return false, errors.New("expired token")
	}

	return true, nil
}
