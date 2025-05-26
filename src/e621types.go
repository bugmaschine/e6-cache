package main

import "time"

type Post struct {
	ID            int           `json:"id"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	File          File          `json:"file"`
	Preview       Preview       `json:"preview"`
	Sample        Sample        `json:"sample"`
	Score         Score         `json:"score"`
	Tags          Tags          `json:"tags"`
	LockedTags    []string      `json:"locked_tags"`
	ChangeSeq     int           `json:"change_seq"`
	Flags         Flags         `json:"flags"`
	Rating        string        `json:"rating"` // s, q or e
	FavCount      int           `json:"fav_count"`
	Sources       []string      `json:"sources"`
	Pools         []int         `json:"pools"`
	Relationships Relationships `json:"relationships"`
	ApproverID    *int          `json:"approver_id"` // nullable
	UploaderID    int           `json:"uploader_id"`
	Description   string        `json:"description"`
	CommentCount  int           `json:"comment_count"`
	IsFavorited   *bool         `json:"is_favorited,omitempty"` // nullable, only if auth provided
}

type PostsResponse struct {
	Posts []Post `json:"posts"`
}

type PostResponse struct {
	Post Post `json:"post"`
}

type CommentsResponse struct {
	Comments []Comment `json:"comments"`
}

type Comment struct {
	ID            int64     `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	PostID        int64     `json:"post_id"`
	CreatorID     int64     `json:"creator_id"`
	Body          string    `json:"body"`
	Score         int       `json:"score"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdaterID     int64     `json:"updater_id"`
	DoNotBumpPost bool      `json:"do_not_bump_post"`
	IsHidden      bool      `json:"is_hidden"`
	IsSticky      bool      `json:"is_sticky"`
	WarningType   *string   `json:"warning_type"`
	WarningUserID *int64    `json:"warning_user_id"`
	CreatorName   string    `json:"creator_name"`
	UpdaterName   string    `json:"updater_name"`
}

type File struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Ext    string `json:"ext"`
	Size   int    `json:"size"`
	MD5    string `json:"md5"`
	URL    string `json:"url"`
}

type Preview struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type Sample struct {
	Has    bool   `json:"has"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type Score struct {
	Up    int `json:"up"`
	Down  int `json:"down"`
	Total int `json:"total"`
}

type Tags struct {
	General   []string `json:"general"`
	Species   []string `json:"species"`
	Character []string `json:"character"`
	Artist    []string `json:"artist"`
	Invalid   []string `json:"invalid"`
	Lore      []string `json:"lore"`
	Meta      []string `json:"meta"`
}

type Flags struct {
	Pending      bool `json:"pending"`
	Flagged      bool `json:"flagged"`
	NoteLocked   bool `json:"note_locked"`
	StatusLocked bool `json:"status_locked"`
	RatingLocked bool `json:"rating_locked"`
	Deleted      bool `json:"deleted"`
}

type Relationships struct {
	ParentID          *int  `json:"parent_id"` // nullable
	HasChildren       bool  `json:"has_children"`
	HasActiveChildren bool  `json:"has_active_children"`
	Children          []int `json:"children"`
}

type Pool struct {
	ID          int       `db:"id"`
	Name        string    `db:"name"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	CreatorID   int       `db:"creator_id"`
	CreatorName string    `db:"creator_name"`
	Description string    `db:"description"`
	IsActive    bool      `db:"is_active"`
	Category    string    `db:"category"`
	PostCount   int       `db:"post_count"`
	PostIDs     []int     // not in DB directly
}
