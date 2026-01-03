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
  name = coalesce(sqlc.arg(name), name),
  description = coalesce(sqlc.arg(description), description),
  published_at = case
    when cast(sqlc.arg(clear_published_at) as boolean) then null
    else coalesce(sqlc.arg(published_at), published_at)
  end,
  total_price_cents = coalesce(sqlc.arg(total_price_cents), total_price_cents),
  location = coalesce(sqlc.arg(location), location),
  starts_at = coalesce(sqlc.arg(starts_at), starts_at),
  duration_minutes = coalesce(nullif(cast(sqlc.arg(duration_minutes) as integer), 0), duration_minutes),
  max_players = coalesce(nullif(cast(sqlc.arg(max_players) as integer), 0), max_players),
  max_waitlist_size = coalesce(sqlc.arg(max_waitlist_size), max_waitlist_size),
  max_guests_per_player = coalesce(sqlc.arg(max_guests_per_player), max_guests_per_player),
  updated_at = current_timestamp
where id = sqlc.arg(id);

-- name: GameListByUser :many
select
  games.id,
  games.name,
  games.location,
  games.starts_at,
  games.published_at,
  games.updated_at,
  games.organizer_id = sqlc.arg(user_id) as is_organizer
from games
left join game_participants
  on games.id = game_participants.game_id and game_participants.user_id = sqlc.arg(user_id)
where games.organizer_id = sqlc.arg(user_id) or game_participants.user_id is not null
order by coalesce(games.published_at, games.updated_at) desc
limit sqlc.arg(limit) offset sqlc.arg(offset);

-- name: GameCountByUser :one
select count(*)
from games
left join game_participants
  on games.id = game_participants.game_id and game_participants.user_id = sqlc.arg(user_id)
where games.organizer_id =sqlc.arg(user_id) or game_participants.user_id is not null;
