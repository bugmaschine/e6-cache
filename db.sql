-- Create table
CREATE TABLE posts (
  id                 BIGINT          PRIMARY KEY,
  created_at         TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at         TIMESTAMP WITH TIME ZONE NOT NULL,
  -- File group
  file_width         INTEGER         NOT NULL,
  file_height        INTEGER         NOT NULL,
  file_ext           TEXT            NOT NULL,
  file_size          BIGINT          NOT NULL,
  file_md5           TEXT            NOT NULL,
  file_url           TEXT            NOT NULL,
  -- Preview group
  preview_width      INTEGER         NOT NULL,
  preview_height     INTEGER         NOT NULL,
  preview_url        TEXT            NOT NULL,
  -- Sample group
  sample_has         BOOLEAN         NOT NULL,
  sample_width       INTEGER         NOT NULL,
  sample_height      INTEGER         NOT NULL,
  sample_url         TEXT            NOT NULL,
  -- Score group
  score_up           INTEGER         NOT NULL,
  score_down         INTEGER         NOT NULL,
  score_total        INTEGER         NOT NULL,
  -- Tags
  tags_general       TEXT[]          NOT NULL,
  tags_species       TEXT[]          NOT NULL,
  tags_character     TEXT[]          NOT NULL,
  tags_artist        TEXT[]          NOT NULL,
  tags_invalid       TEXT[]          NOT NULL,
  tags_lore          TEXT[]          NOT NULL,
  tags_meta          TEXT[]          NOT NULL,
  locked_tags        TEXT[]          NOT NULL,
  -- Other fields
  change_seq         BIGINT          NOT NULL,
  flags_pending      BOOLEAN         NOT NULL,
  flags_flagged      BOOLEAN         NOT NULL,
  flags_note_locked  BOOLEAN         NOT NULL,
  flags_status_locked BOOLEAN        NOT NULL,
  flags_rating_locked BOOLEAN        NOT NULL,
  flags_deleted      BOOLEAN         NOT NULL,
  rating             TEXT            NOT NULL,
  fav_count          INTEGER         NOT NULL,
  sources            TEXT[]          NOT NULL,
  pools              BIGINT[]        NOT NULL,
  -- Relationships
  parent_id          BIGINT,
  has_children       BOOLEAN         NOT NULL,
  has_active_children BOOLEAN        NOT NULL,
  children           BIGINT[]        NOT NULL,
  approver_id        BIGINT,
  uploader_id        BIGINT          NOT NULL,
  description        TEXT            NOT NULL,
  comment_count      INTEGER         NOT NULL,
  is_favorited       BOOLEAN,
  -- Indexes
  UNIQUE (id)
);

CREATE TABLE pools (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    creator_id INTEGER NOT NULL,
    creator_name TEXT,
    description TEXT,
    is_active BOOLEAN NOT NULL,
    category TEXT,
    post_count INTEGER
);

CREATE TABLE pool_posts (
    pool_id INTEGER NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    post_id INTEGER NOT NULL,
    PRIMARY KEY (pool_id, post_id)
);

CREATE TABLE comments (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    post_id BIGINT NOT NULL,
    creator_id BIGINT NOT NULL,
    body TEXT NOT NULL,
    score INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    updater_id BIGINT NOT NULL,
    do_not_bump_post BOOLEAN NOT NULL,
    is_hidden BOOLEAN NOT NULL,
    is_sticky BOOLEAN NOT NULL,
    warning_type TEXT,
    warning_user_id BIGINT,
    creator_name TEXT NOT NULL,
    updater_name TEXT NOT NULL
);
