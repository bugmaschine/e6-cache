package main

import (
	"bugmaschine/e6-cache/logging"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type DB struct {
	db *sql.DB
}

func newDB(server, name, user, password string, port int) (DB, error) {
	dbURI := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		server, port, user, password, name,
	)

	dbConn, err := sql.Open("postgres", dbURI)
	if err != nil {
		return DB{}, err
	}
	if err = dbConn.Ping(); err != nil {
		return DB{}, err
	}
	return DB{db: dbConn}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) CreatePost(ctx context.Context, p *Post) error {

	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.CreatedAt
	}

	query := `
	INSERT INTO posts (
		id, created_at, updated_at,
		file_width, file_height, file_ext, file_size, file_md5, file_url,
		preview_width, preview_height, preview_url,
		sample_has, sample_width, sample_height, sample_url,
		score_up, score_down, score_total,
		tags_general, tags_species, tags_character, tags_artist, tags_invalid, tags_lore, tags_meta,
		locked_tags, change_seq,
		flags_pending, flags_flagged, flags_note_locked, flags_status_locked, flags_rating_locked, flags_deleted,
		rating, fav_count, sources, pools,
		parent_id, has_children, has_active_children, children,
		approver_id, uploader_id, description, comment_count, is_favorited
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
		$20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35,
		$36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47
	)
	`
	_, err := d.db.ExecContext(
		ctx,
		query,
		p.ID, p.CreatedAt, p.UpdatedAt,
		p.File.Width, p.File.Height, p.File.Ext, p.File.Size, p.File.MD5, p.File.URL,
		p.Preview.Width, p.Preview.Height, p.Preview.URL,
		p.Sample.Has, p.Sample.Width, p.Sample.Height, p.Sample.URL,
		p.Score.Up, p.Score.Down, p.Score.Total,
		pq.Array(p.Tags.General), pq.Array(p.Tags.Species), pq.Array(p.Tags.Character),
		pq.Array(p.Tags.Artist), pq.Array(p.Tags.Invalid), pq.Array(p.Tags.Lore), pq.Array(p.Tags.Meta),
		pq.Array(p.LockedTags), p.ChangeSeq,
		p.Flags.Pending, p.Flags.Flagged, p.Flags.NoteLocked, p.Flags.StatusLocked, p.Flags.RatingLocked, p.Flags.Deleted,
		p.Rating, p.FavCount, pq.Array(p.Sources), pq.Array(p.Pools),
		p.Relationships.ParentID, p.Relationships.HasChildren, p.Relationships.HasActiveChildren, pq.Array(p.Relationships.Children),
		p.ApproverID, p.UploaderID, p.Description, p.CommentCount, p.IsFavorited,
	)

	if err != nil {
		logging.Error("Error inserting post: ", err)
	}
	return err
}

func (d *DB) CheckAndInsertPost(ctx context.Context, p *Post) error {
	logging.Info("Checking if post exists: ", p.ID)
	const existsQuery = `SELECT 1 FROM posts WHERE id = $1`
	row := d.db.QueryRowContext(ctx, existsQuery, p.ID)
	var dummy int
	err := row.Scan(&dummy)
	switch {
	case err == sql.ErrNoRows:
		logging.Info("Post does not exist, inserting: ", p.ID)
		return d.CreatePost(ctx, p)
	case err != nil:
		log.Println("Error checking post existence: ", err)
		return err
	default:
		// Row exists, nothing to do
		return nil
	}
}

func (d *DB) SaveComments(comments []Comment) error {
	const query = `
		INSERT INTO comments (
			id, created_at, post_id, creator_id, body, score,
			updated_at, updater_id, do_not_bump_post, is_hidden, is_sticky,
			warning_type, warning_user_id, creator_name, updater_name
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15
		)
		ON CONFLICT (id) DO NOTHING
	`

	stmt, err := d.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range comments {
		_, err := stmt.Exec(
			c.ID,
			c.CreatedAt,
			c.PostID,
			c.CreatorID,
			c.Body,
			c.Score,
			c.UpdatedAt,
			c.UpdaterID,
			c.DoNotBumpPost,
			c.IsHidden,
			c.IsSticky,
			c.WarningType,
			c.WarningUserID,
			c.CreatorName,
			c.UpdaterName,
		)
		if err != nil {
			logging.Error(err.Error())
			return err
		}
	}

	return nil
}

func (d *DB) GetPost(ctx context.Context, id int64) (*Post, error) {
	query := `
	SELECT
		id, created_at, updated_at,
		file_width, file_height, file_ext, file_size, file_md5, file_url,
		preview_width, preview_height, preview_url,
		sample_has, sample_width, sample_height, sample_url,
		score_up, score_down, score_total,
		tags_general, tags_species, tags_character, tags_artist, tags_invalid, tags_lore, tags_meta,
		locked_tags, change_seq,
		flags_pending, flags_flagged, flags_note_locked, flags_status_locked, flags_rating_locked, flags_deleted,
		rating, fav_count, sources, pools,
		parent_id, has_children, has_active_children, children,
		approver_id, uploader_id, description, comment_count, is_favorited
	FROM posts WHERE id = $1
	`
	row := d.db.QueryRowContext(ctx, query, id)
	p := &Post{}

	err := row.Scan(
		&p.ID, &p.CreatedAt, &p.UpdatedAt,
		&p.File.Width, &p.File.Height, &p.File.Ext, &p.File.Size, &p.File.MD5, &p.File.URL,
		&p.Preview.Width, &p.Preview.Height, &p.Preview.URL,
		&p.Sample.Has, &p.Sample.Width, &p.Sample.Height, &p.Sample.URL,
		&p.Score.Up, &p.Score.Down, &p.Score.Total,
		pq.Array(&p.Tags.General), pq.Array(&p.Tags.Species), pq.Array(&p.Tags.Character),
		pq.Array(&p.Tags.Artist), pq.Array(&p.Tags.Invalid), pq.Array(&p.Tags.Lore), pq.Array(&p.Tags.Meta),
		pq.Array(&p.LockedTags), &p.ChangeSeq,
		&p.Flags.Pending, &p.Flags.Flagged, &p.Flags.NoteLocked, &p.Flags.StatusLocked, &p.Flags.RatingLocked, &p.Flags.Deleted,
		&p.Rating, &p.FavCount, pq.Array(&p.Sources), pq.Array(&p.Pools),
		&p.Relationships.ParentID, &p.Relationships.HasChildren, &p.Relationships.HasActiveChildren, pq.Array(&p.Relationships.Children),
		&p.ApproverID, &p.UploaderID, &p.Description, &p.CommentCount, &p.IsFavorited,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (d *DB) UpdatePost(ctx context.Context, p *Post) error {
	p.UpdatedAt = time.Now()

	query := `
	UPDATE posts SET
		created_at = $2, updated_at = $3,
		file_width = $4, file_height = $5, file_ext = $6, file_size = $7, file_md5 = $8, file_url = $9,
		preview_width = $10, preview_height = $11, preview_url = $12,
		sample_has = $13, sample_width = $14, sample_height = $15, sample_url = $16,
		score_up = $17, score_down = $18, score_total = $19,
		tags_general = $20, tags_species = $21, tags_character = $22, tags_artist = $23,
		tags_invalid = $24, tags_lore = $25, tags_meta = $26,
		locked_tags = $27, change_seq = $28,
		flags_pending = $29, flags_flagged = $30, flags_note_locked = $31, flags_status_locked = $32,
		flags_rating_locked = $33, flags_deleted = $34,
		rating = $35, fav_count = $36, sources = $37, pools = $38,
		parent_id = $39, has_children = $40, has_active_children = $41, children = $42,
		approver_id = $43, uploader_id = $44, description = $45, comment_count = $46, is_favorited = $47
	WHERE id = $1
	`
	_, err := d.db.ExecContext(
		ctx,
		query,
		p.ID, p.CreatedAt, p.UpdatedAt,
		p.File.Width, p.File.Height, p.File.Ext, p.File.Size, p.File.MD5, p.File.URL,
		p.Preview.Width, p.Preview.Height, p.Preview.URL,
		p.Sample.Has, p.Sample.Width, p.Sample.Height, p.Sample.URL,
		p.Score.Up, p.Score.Down, p.Score.Total,
		pq.Array(p.Tags.General), pq.Array(p.Tags.Species), pq.Array(p.Tags.Character),
		pq.Array(p.Tags.Artist), pq.Array(p.Tags.Invalid), pq.Array(p.Tags.Lore), pq.Array(p.Tags.Meta),
		pq.Array(p.LockedTags), p.ChangeSeq,
		p.Flags.Pending, p.Flags.Flagged, p.Flags.NoteLocked, p.Flags.StatusLocked, p.Flags.RatingLocked, p.Flags.Deleted,
		p.Rating, p.FavCount, pq.Array(p.Sources), pq.Array(p.Pools),
		p.Relationships.ParentID, p.Relationships.HasChildren, p.Relationships.HasActiveChildren, pq.Array(p.Relationships.Children),
		p.ApproverID, p.UploaderID, p.Description, p.CommentCount, p.IsFavorited,
	)
	if err != nil {
		logging.Error("Error updating post: ", err)
	}
	return err
}

// DeletePost removes a post by its ID.
func (d *DB) DeletePost(ctx context.Context, id int64) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, id)
	if err != nil {
		logging.Error("Error deleting post: ", err)
	}
	return err
}

func (d *DB) UpdatePool(ctx context.Context, p *Pool) error {
	query := `
		INSERT INTO pools (
			id, name, created_at, updated_at, creator_id, creator_name,
			description, is_active, category, post_count
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT(id) DO UPDATE SET
			name = EXCLUDED.name,
			created_at = EXCLUDED.created_at,
			updated_at = EXCLUDED.updated_at,
			creator_id = EXCLUDED.creator_id,
			creator_name = EXCLUDED.creator_name,
			description = EXCLUDED.description,
			is_active = EXCLUDED.is_active,
			category = EXCLUDED.category,
			post_count = EXCLUDED.post_count
	`

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, query,
		p.ID, p.Name, p.CreatedAt, p.UpdatedAt, p.CreatorID, p.CreatorName,
		p.Description, p.IsActive, p.Category, p.PostCount,
	)
	if err != nil {
		logging.Error("error upserting pool: ", err)
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM pool_posts WHERE pool_id = $1`, p.ID)
	if err != nil {
		logging.Error("error clearing pool_posts: ", err)
		return err
	}

	for _, postID := range p.PostIDs {
		_, err = tx.ExecContext(ctx, `INSERT INTO pool_posts (pool_id, post_id) VALUES ($1, $2)`, p.ID, postID)
		if err != nil {
			logging.Error("error inserting pool_post: ", err)
			return err
		}
	}

	return tx.Commit()
}

func (d *DB) SearchPosts(ctx context.Context, minScoreTotal int, rating string, tagsToSearch []string, limit, offset int) ([]*Post, error) {
	// Base query
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
	SELECT
		id, created_at, updated_at,
		file_width, file_height, file_ext, file_size, file_md5, file_url,
		preview_width, preview_height, preview_url,
		sample_has, sample_width, sample_height, sample_url,
		score_up, score_down, score_total,
		tags_general, tags_species, tags_character, tags_artist, tags_invalid, tags_lore, tags_meta,
		locked_tags, change_seq,
		flags_pending, flags_flagged, flags_note_locked, flags_status_locked, flags_rating_locked, flags_deleted,
		rating, fav_count, sources, pools,
		parent_id, has_children, has_active_children, children,
		approver_id, uploader_id, description, comment_count, is_favorited
	FROM posts
	WHERE 1=1`)

	args := []any{}
	paramIndex := 1

	if minScoreTotal != 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND score_total >= $%d", paramIndex))
		args = append(args, minScoreTotal)
		paramIndex++
	}

	if rating != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND rating = $%d", paramIndex))
		args = append(args, rating)
		paramIndex++
	}

	if len(tagsToSearch) > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" AND tags_general @> $%d", paramIndex))
		args = append(args, pq.Array(tagsToSearch))
		paramIndex++
	}

	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1))
	args = append(args, limit, offset)

	rows, err := d.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Post
	for rows.Next() {
		p := &Post{}
		err := rows.Scan(
			&p.ID, &p.CreatedAt, &p.UpdatedAt,
			&p.File.Width, &p.File.Height, &p.File.Ext, &p.File.Size, &p.File.MD5, &p.File.URL,
			&p.Preview.Width, &p.Preview.Height, &p.Preview.URL,
			&p.Sample.Has, &p.Sample.Width, &p.Sample.Height, &p.Sample.URL,
			&p.Score.Up, &p.Score.Down, &p.Score.Total,
			pq.Array(&p.Tags.General), pq.Array(&p.Tags.Species), pq.Array(&p.Tags.Character),
			pq.Array(&p.Tags.Artist), pq.Array(&p.Tags.Invalid), pq.Array(&p.Tags.Lore), pq.Array(&p.Tags.Meta),
			pq.Array(&p.LockedTags), &p.ChangeSeq,
			&p.Flags.Pending, &p.Flags.Flagged, &p.Flags.NoteLocked, &p.Flags.StatusLocked, &p.Flags.RatingLocked, &p.Flags.Deleted,
			&p.Rating, &p.FavCount, pq.Array(&p.Sources), pq.Array(&p.Pools),
			&p.Relationships.ParentID, &p.Relationships.HasChildren, &p.Relationships.HasActiveChildren, pq.Array(&p.Relationships.Children),
			&p.ApproverID, &p.UploaderID, &p.Description, &p.CommentCount, &p.IsFavorited,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
