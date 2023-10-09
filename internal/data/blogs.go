package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mozillazg/go-slugify"
)

// Blog is the definition of a single blog
type Blog struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	CreatedByID int        `json:"createdby_id"`
	CreatedBy   User       `json:"created_by"`
	Description string     `json:"description"`
	Content     string     `json:"content"`
	Categorys   []Category `json:"category"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CategoryIDs []int      `json:"category_ids,omitempty"`
}

// Category is the definition of a single category type
type Category struct {
	ID           int       `json:"id"`
	CategoryName string    `json:"category_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GetAll returns a slice of all blogs
func (b *Blog) GetAll() ([]*Blog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// (select array_to_string(array_agg(category_id), ',') from blogs_categorys where blog_id = b.id)
	query := `SELECT b.id, b.title, b.slug, b.createdby_id,  b.description, b.content, b.created_at, b.updated_at, 
            u.id, u.first_name
            from blogs b
            left join users u on (b.createdby_id = u.id)
            order by b.title`

	var blogs []*Blog

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var blog Blog
		err := rows.Scan(
			&blog.ID,
			&blog.Title,
			&blog.Slug,
			&blog.CreatedByID,
			&blog.Description,
			&blog.Content,
			&blog.CreatedAt,
			&blog.UpdatedAt,
			&blog.CreatedBy.ID,        //User ID
			&blog.CreatedBy.FirstName, // User firstname
		)
		if err != nil {
			return nil, err
		}

		// get categorys
		categorys, ids, err := b.categorysForBlog(blog.ID)
		if err != nil {
			return nil, err
		}

		blog.Categorys = categorys
		blog.CategoryIDs = ids

		blogs = append(blogs, &blog)
	}

	return blogs, nil
}

// GetAllPaginated returns a slice of all blogs but paginated
func (b *Blog) GetAllPaginated(page, pageSize int) ([]*Blog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	limit := pageSize
	offset := (page - 1) * pageSize

	// (select array_to_string(array_agg(category_id), ',') from blogs_categorys where blog_id = b.id)
	query := `select b.id, b.title, b.slug, b.createdby_id, b.created_by,  b.description, b.content, b.created_at, b.updated_at, 
            u.id, u.first_name, 
            from blogs b
            left join users u on (b.createdby_id = u.id)
            order by b.title
			limit $1 offset $2`

	var blogs []*Blog

	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var blog Blog
		err := rows.Scan(
			&blog.ID,
			&blog.Title,
			&blog.Slug,
			&blog.CreatedByID,
			&blog.CreatedBy,
			&blog.Description,
			&blog.Content,
			&blog.CreatedAt,
			&blog.UpdatedAt,
			&blog.CreatedBy.ID,
			&blog.CreatedBy.FirstName,
		)
		if err != nil {
			return nil, err
		}

		// get categorys
		categorys, ids, err := b.categorysForBlog(blog.ID)
		if err != nil {
			return nil, err
		}
		blog.Categorys = categorys
		blog.CategoryIDs = ids

		blogs = append(blogs, &blog)
	}

	return blogs, nil
}

// GetOneById returns one blog by its id
func (b *Blog) GetOneById(id int) (*Blog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select b.id, b.title, b.slug, b.createdby_id, b.created_by,  b.description, b.content, b.created_at, b.updated_at,
            u.id, u.first_name,
            from blogs b
            left join users u on (b.createdby_id = u.id)
            where b.id = $1`

	row := db.QueryRowContext(ctx, query, id)

	var blog Blog

	err := row.Scan(
		&blog.ID,
		&blog.Title,
		&blog.Slug,
		&blog.CreatedByID,
		&blog.CreatedBy,
		&blog.Description,
		&blog.Content,
		&blog.CreatedAt,
		&blog.UpdatedAt,
		&blog.CreatedBy.ID,
		&blog.CreatedBy.FirstName)
	if err != nil {
		return nil, err
	}

	// get categorys
	categorys, ids, err := b.categorysForBlog(blog.ID)
	if err != nil {
		return nil, err
	}

	blog.Categorys = categorys
	blog.CategoryIDs = ids

	return &blog, nil
}

// GetOneBySlug returns one blog by slug
func (b *Blog) GetOneBySlug(slug string) (*Blog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `SELECT b.id, b.title, b.slug, b.createdby_id, b.description, b.content, b.created_at, b.updated_at, 
			u.id, u.first_name
			from blogs b
			left join users u on (b.createdby_id = u.id)
			where b.slug = $1`

	row := db.QueryRowContext(ctx, query, slug)

	var blog Blog

	err := row.Scan(
		&blog.ID,
		&blog.Title,
		&blog.Slug,
		&blog.CreatedByID,
		&blog.Description,
		&blog.Content,
		&blog.CreatedAt,
		&blog.UpdatedAt,
		&blog.CreatedBy.ID,
		&blog.CreatedBy.FirstName)
	if err != nil {
		return nil, err
	}

	// get categorys
	categorys, ids, err := b.categorysForBlog(blog.ID)
	if err != nil {
		return nil, err
	}

	blog.Categorys = categorys
	blog.CategoryIDs = ids

	return &blog, nil
}

// categorysForBlog returns all categories for a given blog id
func (b *Blog) categorysForBlog(id int) ([]Category, []int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// get genres
	var categorys []Category
	var categoryIDs []int
	categoryQuery := `SELECT id, category_name, created_at, updated_at from categorys where id in (SELECT category_id 
                from blogs_categorys where blog_id = $1) order by category_name`

	cRows, err := db.QueryContext(ctx, categoryQuery, id)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}
	defer cRows.Close()

	var category Category
	for cRows.Next() {

		err = cRows.Scan(
			&category.ID,
			&category.CategoryName,
			&category.CreatedAt,
			&category.UpdatedAt)
		if err != nil {
			return nil, nil, err
		}
		categorys = append(categorys, category)
		categoryIDs = append(categoryIDs, category.ID)
	}

	return categorys, categoryIDs, nil
}

// Create saves one blog to the database
func (b *Blog) Create(blog Blog) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `insert into blogs (title, slug, createdby_id, created_by, description, content, created_at, updated_at)
            values ($1, $2, $3, $4, $5, $6, $7, $8) returning id`

	var newID int
	err := db.QueryRowContext(ctx, stmt,
		blog.Title,
		blog.CreatedByID,
		blog.CreatedBy,
		slugify.Slugify(b.Title),
		blog.Description,
		blog.Content,
		time.Now(),
		time.Now(),
	).Scan(&newID)
	if err != nil {
		return 0, err
	}

	// update categories using caegory ids
	if len(blog.CategoryIDs) > 0 {
		stmt = `DELETE from blogs_categorys WHERE blog_id = $1`
		_, err := db.ExecContext(ctx, stmt, blog.ID)
		if err != nil {
			return newID, fmt.Errorf("blog updated, but categorys not: %s", err.Error())
		}

		for _, x := range blog.CategoryIDs {
			stmt = `insert into blogs_categorys (blog_id, category_id, created_at, updated_at)
			values ($1, $2, $3, $4)`
			_, err = db.ExecContext(ctx, stmt, newID, x, time.Now(), time.Now())
			if err != nil {
				return newID, fmt.Errorf("blog updated, but categorys not: %s", err.Error())
			}
		}
	}

	return newID, nil
}

// Update updates one blog in the database
func (b *Blog) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `update blogs set
        title = $1,
        createdby_id = $2,
        slug = $3,
        description = $4,
		content = $5,
        updated_at = $6
        where id = $7`

	_, err := db.ExecContext(ctx, stmt,
		b.Title,
		b.CreatedByID,
		slugify.Slugify(b.Title),
		b.Description,
		b.Content,
		time.Now(),
		b.ID)
	if err != nil {
		return err
	}

	// update categorys
	if len(b.Categorys) > 0 {
		// delete existing category
		stmt = `delete from categorys where blog_id = $1`
		_, err := db.ExecContext(ctx, stmt, b.ID)
		if err != nil {
			return fmt.Errorf("blog updated, but categorys not: %s", err.Error())
		}

		// add new categorys
		for _, x := range b.Categorys {
			stmt = `insert into blogs_categorys (blog_id, category_id, created_at, updated_at)
                values ($1, $2, $3, $4)`
			_, err = db.ExecContext(ctx, stmt, b.ID, x.ID, time.Now(), time.Now())
			if err != nil {
				return fmt.Errorf("blog updated, but categorys not: %s", err.Error())
			}
		}
	}

	return nil
}

// DeleteByID deletes a blog by id
func (b *Blog) DeleteByID(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `delete from blogs where id = $1`
	_, err := db.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}
	return nil
}

// Get blogs by the creator Id
func (u *User) FindBlogByUserId(id int) ([]*Blog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `SELECT title, slug, description, content, created_at, updated_at FROM blog WHERE id = ? AND author = ?, id, user.ID`

	// Query every row for blogs and close when  done
	rows, err := db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// varibale user points at an slice of User. we will be returning every field in that struct
	var blogs []*Blog

	// for loop to each individual field in our User slice one by one so they can be individually scanned for errors before we return any of them
	for rows.Next() {
		var blog Blog
		err := rows.Scan(
			&blog.Title,
			&blog.Slug,
			&blog.CreatedByID,
			&blog.CreatedBy,
			&blog.Description,
			&blog.Content,
			&blog.CreatedAt,
			&blog.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// if no errors are found each field will be appended back to the slice blog
		blogs = append(blogs, &blog)
	}
	return blogs, nil

}
