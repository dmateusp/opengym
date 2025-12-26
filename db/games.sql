-- name: GameCreate :one
insert into games(
  id,
  organizer_id,
  name,
  description,
  published_at,
  total_price_cents,
  location,
  starts_at,
  duration_minutes,
  max_players,
  max_waitlist_size,
  max_guests_per_player
) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: GameGetById :one
select * from games where id = ?;

-- name: GameUpdate :exec
update games
set
  name = coalesce(?, name),
  description = coalesce(?, description),
  published_at = coalesce(?, published_at),
  total_price_cents = coalesce(?, total_price_cents),
  location = coalesce(?, location),
  starts_at = coalesce(?, starts_at),
  duration_minutes = coalesce(?, duration_minutes),
  max_players = coalesce(?, max_players),
  max_waitlist_size = coalesce(?, max_waitlist_size),
  max_guests_per_player = coalesce(?, max_guests_per_player),
  updated_at = current_timestamp
where id = ?;
