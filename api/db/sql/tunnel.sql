-- name: CreateTunnel :one
insert into tunnels (name, domain, target, protocol, ipv4, id, user_id, updated_at)
values (?, ?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: GetTunnel :one
select *
from tunnels
where user_id = ? and id = ?;

-- name: UpdateTunnel :one
update tunnels
set 
  name = COALESCE(sqlc.narg('name'), name),
  domain = COALESCE(sqlc.narg('domain'), domain), 
  target = COALESCE(sqlc.narg('target'), target), 
  protocol = COALESCE(sqlc.narg('protocol'), protocol),
  ipv4 = COALESCE(sqlc.narg('ipv4'), ipv4),
  updated_at = current_timestamp
where id = ?
returning *;

-- name: DeleteTunnel :one
delete from tunnels
where id = ? and user_id = ?
returning *;

-- name: GetTunnels :many
select *
from tunnels
where user_id = ?
order by domain;
