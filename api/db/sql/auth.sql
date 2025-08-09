-- name: CreateRefreshToken :exec
insert into refresh_tokens (
  id,
  user_id
) values (?, ?);

-- name: GetRefreshToken :one
select id, user_id
from refresh_tokens
where id = ? and user_id = ?;

-- name: DeleteRefreshToken :exec
delete from refresh_tokens
where id = ? and user_id = ?;

-- name: DeleteRefreshTokensByUserID :exec
delete from refresh_tokens
where user_id = ?;
