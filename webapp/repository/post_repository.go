package repository

import (
	"errors"
	"math"
	"time"
)

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

type Filter struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	OrderBy  string `json:"order_by"`
	Query    string `json:"query"`
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

type PostRepository interface {
	CreatePost(title, url string, userID int) (int, error)
	AddComment(userID, postID int, body string) (int, error)
	AddVote(userID, postID int, body int) error
	GetAll(filter Filter) ([]Post, Metadata, error)
	GetByID(id int) (*Post, error)
	GetComments(postID int) ([]Comment, error)
}
