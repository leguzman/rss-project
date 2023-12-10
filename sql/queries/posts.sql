-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, description, published_at, url, feed_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
-- name: GetUserPosts :many
SELECT posts.* FROM posts
JOIN feed_follows ON feed_follows.feed_id = posts.feed_id
WHERE feed_follows.user_id=$1
ORDER BY posts.published_at
LIMIT $2;

-- name: FilterUserPosts :many
SELECT posts.* FROM posts
JOIN feed_follows ON feed_follows.feed_id = posts.feed_id
WHERE feed_follows.user_id=$1
AND (@title::text = '' OR posts.title ILIKE '%' || @title || '%')
AND (@description::text = '' OR posts.description ILIKE '%' || @description || '%')
AND (@before::TIMESTAMP = '0001-01-01' OR posts.published_at <= @before )
AND (@after::TIMESTAMP = '0001-01-01' OR posts.published_at >= @after )
ORDER BY posts.published_at
LIMIT $2;
