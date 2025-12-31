-- name: UserUpsertRetuningId :one
insert into users(
    name,
    email,
    photo,
    is_demo,
    updated_at
) values (?, ?, ?, ?, current_timestamp)
on conflict(email) do update set
    name = excluded.name,
    photo = coalesce(excluded.photo, users.photo), -- only update the photo if the new value is not null
    updated_at = excluded.updated_at
returning id;

-- name: UserGetById :one
select sqlc.embed(users)
from users
where id = ?
limit 1;

-- name: ListDemoUsers :many
select sqlc.embed(users)
from users
where is_demo;
