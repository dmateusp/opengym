-- name: ParticipantsUpsert :exec
insert into game_participants(
    user_id,
    game_id,
    going,
    going_updated_at,
    confirmed_at
) values (?, ?, ?, current_timestamp, ?)
on conflict(user_id, game_id) do update set
    updated_at = current_timestamp,
    going = coalesce(excluded.going, game_participants.going),
    going_updated_at = iif(excluded.going = game_participants.going, game_participants.going_updated_at, current_timestamp),
    confirmed_at = coalesce(excluded.confirmed_at, game_participants.confirmed_at);

-- name: ParticipantsList :many
select
    users.id = sqlc.arg(organizer_id) as is_organizer,
    sqlc.embed(game_participants),
    sqlc.embed(users)
from game_participants
join users on game_participants.user_id = users.id
where game_participants.game_id = sqlc.arg(game_id)
order by
    1 desc, -- if the user is the organizer, they should have priority
    game_participants.going_updated_at asc
