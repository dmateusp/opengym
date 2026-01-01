-- name: ParticipantsUpsert :exec
insert into game_participants(
    user_id,
    game_id,
    created_at,
    updated_at,
    going,
    confirmed
) values (?, ?, current_timestamp, current_timestamp, ?, ?)
on conflict(user_id, game_id) do update set
    updated_at = current_timestamp,
    going = coalesce(excluded.going, game_participants.going),
    confirmed = coalesce(excluded.confirmed, game_participants.confirmed);

-- name: ParticipantsList :many
select
    sqlc.embed(game_participants),
    sqlc.embed(users),
    -- we need the following fields to figure out the participation status
    games.max_players,
    games.max_waitlist_size
from game_participants
join games on game_participants.game_id = games.id
join users on game_participants.user_id = users.id
where games.id = ?
order by game_participants.updated_at asc
