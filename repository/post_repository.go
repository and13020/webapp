package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

type PostRepository interface {
	CreatePost(title, url string, userID int) (int, error)
	AddComment(userID, postID int, body string) (int, error)
	AddVote(userID, postID int) error
	GetAll(filter Filter) ([]Post, Metadata, error)
	GetByID(id int) (*Post, error)
	GetComments(postID int) ([]Comment, error)
}

type Filter struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	OrderBy  string `json:"order_by"`
	Query    string `json:"query"`
}

type Post struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	UserID       int       `json:"user_id"`
	UserName     string    `json:"user_name"`
	CreatedAt    time.Time `json:"created_at"`
	CommentCount int       `json:"comment_count"`
	VoteCount    int       `json:"vote_count"`
	TotalRecords int       `json:"total_records"`
}

type Comment struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	UserID    int       `json:"user_id"`
	PostID    int       `json:"post_id"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
}

func (f *Filter) Validate() error {
	if f.PageSize <= 0 || f.PageSize >= 100 {
		return errors.New("Invalid page range: 1 to 100")
	}
	return nil
}

type Metadata struct {
	CurrentPage  int `json:"current_page"`
	PageSize     int `json:"page_size"`
	FirstPage    int `json:"first_page"`
	NextPage     int `json:"next_page"`
	PrevPage     int `json:"prev_page"`
	LastPage     int `json:"last_page"`
	TotalRecords int `json:"total_records"`
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	meta := Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}

	meta.NextPage = meta.CurrentPage + 1
	meta.PrevPage = meta.CurrentPage - 1
	if meta.CurrentPage <= meta.FirstPage {
		meta.PrevPage = 0
	}

	if meta.CurrentPage >= meta.NextPage {
		meta.NextPage = 0
	}

	return meta
}

type SQLPostRepository struct {
	DB *sql.DB
}

func NewSQLPostRepository(db *sql.DB) *SQLPostRepository {
	return &SQLPostRepository{DB: db}
}

var (
	ErrDuplicatePostTitle = errors.New("duplicate post title")
	ErrDuplicateVote      = errors.New("duplicate vote")
)

func (r *SQLPostRepository) CreatePost(title, url string, userID int) (int, error) {
	stmt := "INSERT INTO posts (title, url, user_id) VALUES (?, ?, ?)"

	result, err := r.DB.Exec(stmt, title, url, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: posts.title") {
			return 0, ErrDuplicatePostTitle
		}
		return 0, errors.New("Could not execute 'INSERT INTO posts...' statement")
	}

	postID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(postID), err
}

func (r *SQLPostRepository) AddComment(userID, postID int, body string) (int, error) {
	stmt := "INSERT INTO comments (user_id, post_id, body) VALUES (?, ?, ?)"
	result, err := r.DB.Exec(stmt, userID, postID, body)
	if err != nil {
		return 0, err
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(commentID), nil
}

func (r *SQLPostRepository) AddVote(userID, postID int) error {
	stmt := "INSERT INTO votes (user_id, post_id) VALUES (?, ?)"

	_, err := r.DB.Exec(stmt, userID, postID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "PRIMARY KEY constraint failed") {
			return ErrDuplicateVote
		}
		return err
	}

	return nil
}

// GetAll accepts any filter criteria for pagination and then returns a slice of all posts, metadata and an error if applicable.
// If the filter is not valid, nil is returned with the error.
func (r *SQLPostRepository) GetAll(filter Filter) ([]Post, Metadata, error) {
	if err := filter.Validate(); err != nil {
		return nil, Metadata{}, err
	}

	baseQuery := `
		SELECT 
			COUNT(*) OVER() as total_records, 
			p.id, p.title, p.url, p.user_id, p.created_at, 
			u.name as user_name, 
			COUNT(DISTINCT c.id) as comment_count, 
			COUNT(DISTINCT v.user_id) as vote_count 
		FROM posts p 
		LEFT JOIN users u ON p.user_id = u.id 
		LEFT JOIN comments c ON p.id = c.post_id 
		LEFT JOIN votes v ON p.id = v.post_id 
	`

	var args []any

	if filter.Query != "" {
		baseQuery += " WHERE LOWER(p.title) LIKE ?"
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
	}

	baseQuery += " GROUP BY p.id, p.title, p.url, p.user_id, p.created_at, u.name"
	if filter.OrderBy == "popular" {
		baseQuery += " ORDER BY vote_count DESC, p.created_at DESC"
	} else {
		baseQuery += " ORDER BY p.created_at DESC"
	}

	limit := filter.PageSize
	offset := (filter.Page - 1) * filter.PageSize
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	var posts []Post
	var totalRecords int
	for rows.Next() {
		var post Post
		err := rows.Scan(&totalRecords,
			&post.ID,
			&post.Title,
			&post.URL,
			&post.UserID,
			&post.CreatedAt,
			&post.UserName,
			&post.CommentCount,
			&post.VoteCount)
		if err != nil {
			return nil, Metadata{}, err
		}

		post.TotalRecords = totalRecords
		posts = append(posts, post)
	}

	// If rows had an error during iteration
	if err := rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	// If nothing returned
	if len(posts) == 0 {
		return []Post{}, Metadata{}, nil
	}

	metadata := calculateMetadata(totalRecords, filter.Page, filter.PageSize)
	return posts, metadata, nil
}

// GetByID accepts an id and will return a Post object with the given posts.id field if it exists
func (r *SQLPostRepository) GetByID(id int) (*Post, error) {
	stmt := `
	SELECT p.id, p.title, p.url, p.user_id, p.created_at,
	u.name as user_name,
	COUNT(DISTINCT c.id) AS comment_count,
	COUNT(DISTINCT v.user_id) AS vote_count 
	FROM posts p
	LEFT JOIN users u ON p.user_id = u.id
	LEFT JOIN comments c ON p.id = c.post_id 
	LEFT JOIN votes v ON p.id = v.post_id 
	WHERE p.id = ?
	GROUP BY p.id, p.title, p.url, p.user_id, p.created_at, u.name
	`

	row := r.DB.QueryRow(stmt, id)
	var post Post
	err := row.Scan(&post.ID,
		&post.Title,
		&post.URL,
		&post.UserID,
		&post.CreatedAt,
		&post.UserName,
		&post.CommentCount,
		&post.VoteCount)

	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *SQLPostRepository) GetComments(postID int) ([]Comment, error) {
	query := `SELECT c.id, c.body, c.user_id, c.post_id, c.created_at, u.name as user_name
		FROM comments c 
		LEFT JOIN users u ON c.user_id = u.id 
		WHERE c.post_id = ? 
		ORDER BY c.created_at ASC
	`

	rows, err := r.DB.Query(query, postID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID,
			&comment.Body,
			&comment.UserID,
			&comment.PostID,
			&comment.CreatedAt,
			&comment.UserName)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return []Comment{}, nil // we return []Comment{} rather than nil to indicate we found 0 results
	}

	return comments, nil

}

func (p *Post) GetVoteCountsHuman() string {
	return fmt.Sprintf("%d votes", p.VoteCount)
}

func (p *Post) GetCommentCountsHuman() string {
	return fmt.Sprintf("%d comments", p.CommentCount)
}

func (p *Post) CreatedAtHuman() string {
	d := time.Since(p.CreatedAt)
	return durationSincePost(d)
}

func durationSincePost(d time.Duration) string {
	// n = hour/day/second/year
	const hrsInYear = 8760
	hrs := int(d.Hours())

	// Years
	if mod := hrs / hrsInYear; mod >= 1 {
		if mod > 1 {
			return fmt.Sprintf("%d years ago", mod)
		}
		return fmt.Sprintf("%d year ago", mod)
	}

	// Months
	if mod := hrs / (hrsInYear / 12); mod >= 1 {
		if mod > 1 {
			return fmt.Sprintf("%d months ago", mod)
		}
		return fmt.Sprintf("%d month ago", mod)
	}

	// Days
	if mod := hrs / (hrsInYear / 365); mod >= 1 {
		if mod > 1 {
			return fmt.Sprintf("%d days ago", mod)
		}
		return fmt.Sprintf("%d day ago", mod)
	}

	// Hours
	if hrs > 1 {
		return fmt.Sprintf("%d hours ago", hrs)
	}
	if hrs == 1 {
		return fmt.Sprintf("%d hour ago", hrs)
	}

	// Minutes
	mins := int(d.Minutes())
	if mins > 1 {
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if mins == 1 {
		return fmt.Sprintf("%d minute ago", mins)
	}

	// Seconds
	secs := int(d.Seconds())
	if secs > 1 {
		return fmt.Sprintf("%d seconds ago", secs)
	}

	return fmt.Sprintf("%d second ago", secs)
}

func (p *Post) GetHost() string {
	url, err := url.Parse(p.URL)
	if err != nil {
		return "<invalid-host>"
	}
	return url.Hostname()
}
