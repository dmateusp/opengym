-- name: ParticipantsUpsert :exec
insert into game_participants(
    user_id,
    game_id,
    going,
    going_updated_at,
    confirmed_at,
    guests,
    reimbursement_reference
) values (?, ?, ?, ?, ?, ?, sqlc.narg(reimbursement_reference))
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
    game_participants.going_updated_at asc,
    -- Ties on going_updated_at are common in tests (static clocks) and can happen in
    -- production too; ordering by the auto-incremented participant ID makes queue
    -- ordering deterministic without relying on timestamp precision.
    game_participants.rowid asc;

-- name: ParticipantGetByGameAndUser :one
select *
from game_participants
where game_id = sqlc.arg(game_id)
    and user_id = sqlc.arg(user_id);

-- name: ReimbursementsListByGame :many
select
    sqlc.embed(users),
    game_participants.reimbursement_reference,
    game_participants.reimbursed_at,
    game_participants.reimbursement_received_at
from game_participants
join users on game_participants.user_id = users.id
where game_participants.game_id = sqlc.arg(game_id)
    and game_participants.going = true
order by
    game_participants.going_updated_at asc;

-- name: ParticipantUpdateReimbursedAt :execrows
update game_participants
set
    updated_at = current_timestamp,
    reimbursed_at = sqlc.arg(reimbursed_at)
where game_id = sqlc.arg(game_id)
    and user_id = sqlc.arg(user_id);

-- name: ParticipantUpdateReimbursementReceivedAt :execrows
update game_participants
set
    updated_at = current_timestamp,
    reimbursement_received_at = sqlc.arg(reimbursement_received_at)
where game_id = sqlc.arg(game_id)
    and user_id = sqlc.arg(user_id);
