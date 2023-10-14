package data

import "testing"

func Test_Ping(t *testing.T) {
	err := testDB.Ping()
	if err != nil {
		t.Error("failed to ping the database")
	}
}

func TestBlog_GetAll(t *testing.T) {
	all, err := models.Blog.GetAll()
	if err != nil {
		t.Error("failed to get all blogs", err)
	}

	if len(all) != 1 {
		t.Error("failed to get the correct number of blogs")
	}
}

func TestBlog_GetOneByID(t *testing.T) {
	b, err := models.Blog.GetOneById(1)
	if err != nil {
		t.Error("failed to get one blog by id", err)
	}

	if b.Title != "My Blog" {
		t.Errorf("expected title to be My Blog but got %s", b.Title)
	}

}

func TestBlog_GetOneBySlug(t *testing.T) {
	b, err := models.Blog.GetOneBySlug("my-blog")
	if err != nil {
		t.Error("failed to get one blog by slug", err)
	}

	if b.Title != "My Blog" {
		t.Errorf("expected title to be My Blog but got %s", b.Title)
	}
	_, err = models.Blog.GetOneBySlug("bad-slug")
	if err == nil {
		t.Error("did not get an error attempting to fetch non existing slug")
	}

}
