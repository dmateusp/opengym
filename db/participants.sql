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
