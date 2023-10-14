package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApplication_Allusers(t *testing.T) {
	// create some mock fows for our mock db
	var mockedRows = mockDB.NewRows([]string{"id", "email", "first_name", "last_name", "password", "created_at", "updated_at", "has_token"})
	mockedRows.AddRow("1", "me@here.com", "Jack", "Smith", "abc123", time.Now(), time.Now(), 0)

	// tell mock what queries we expect
	mockDB.ExpectQuery("select \\\\* ").WillReturnRows(mockedRows)

	// create a test recorder which satisfies the requirements for a ResponseRecorder
	rr := httptest.NewRecorder()

	// create request
	req, _ := http.NewRequest("POST", "/admin/users", nil)
	// call the handler
	handler := http.HandlerFunc(testApp.AllUsers)
	handler.ServeHTTP(rr, req)

	// check for expected status code
	if rr.Code != http.StatusOK {
		t.Error("AllUsers returned wrong status code of", rr.Code)
	}
}
