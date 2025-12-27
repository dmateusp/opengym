-- name: UserUpsertRetuningId :one
insert into users(
    name,
    email,
    photo,
    updated_at
) values (?, ?, ?, current_timestamp)
on conflict(email) do update set
    name = excluded.name,
    photo = coalesce(excluded.photo, users.photo), -- only update the photo if the new value is not null
    updated_at = excluded.updated_at
returning id;

-- name: UserGetById :one
select id, name, email, photo, created_at, updated_at
from users
where id = ?
limit 1;
