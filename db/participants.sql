-- name: ParticipantsUpsert :exec
insert into game_participants(
    user_id,
    game_id,
    going,
    going_updated_at,
    confirmed_at,
    guests
) values (?, ?, ?, ?, ?, ?)
on conflict(user_id, game_id) do update set
    updated_at = current_timestamp,
    going = coalesce(excluded.going, game_participants.going),
    -- we only update going_updated_at if a field relevant to a participant's order in the queue has changed
    going_updated_at = iif(
        excluded.going = game_participants.going and (excluded.guests is null or excluded.guests is game_participants.guests),
        game_participants.going_updated_at,
        excluded.going_updated_at
    ),
    confirmed_at = coalesce(excluded.confirmed_at, game_participants.confirmed_at),
    guests = coalesce(excluded.guests, game_participants.guests);

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
