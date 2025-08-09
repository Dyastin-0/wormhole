-- name: CreateUser :one
insert into users (id, email, name, username, password)
values (?, ?, ?, ?, ?)
returning *;

-- name: GetUserByID :one
select *
from users
where id = ?;

-- name: GetUserByEmail :one
select *
from users
where email = ?;

-- name: GetUserByEmailAndPassword :one
select *
from users
where email = ? and password = ?;

-- name: UpdateUser :one
update users
set 
  email = COALESCE(sqlc.narg('email'), email),
  name = COALESCE(sqlc.narg('name'), name),
  password = COALESCE(sqlc.narg('password'), password),
  updated_at = current_timestamp
where id = sqlc.narg('id')
returning *;

-- name: DeleteUser :one
delete from users
where id = ?
returning *;
