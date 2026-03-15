-- +goose Up
-- +goose StatementBegin
-- Backfill NULL reimbursement references with random 4-char hex values.
-- This may conflict with the unique index in rare cases; rerun migration on failure.
with generated_refs as (
  select
    rowid,
    upper(substr(hex(randomblob(2)), 1, 4)) as new_ref
  from game_participants
  where reimbursement_reference is null
)
update game_participants
set reimbursement_reference = (
  select new_ref
  from generated_refs
  where generated_refs.rowid = game_participants.rowid
)
where reimbursement_reference is null;

-- Rebuild table to enforce NOT NULL while preserving rowid values used for tie-breaking.
create table game_participants_new (
    user_id integer not null,
    game_id text not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null,
    going_updated_at datetime default current_timestamp not null,
    going boolean default true,
    confirmed_at datetime default current_timestamp,
    guests integer default 0,
    reimbursed_at datetime,
    reimbursement_received_at datetime,
    reimbursement_reference text not null,
    primary key (user_id, game_id)
);

insert into game_participants_new(
    rowid,
    user_id,
    game_id,
    created_at,
    updated_at,
    going_updated_at,
    going,
    confirmed_at,
    guests,
    reimbursed_at,
    reimbursement_received_at,
    reimbursement_reference
)
select
    rowid,
    user_id,
    game_id,
    created_at,
    updated_at,
    going_updated_at,
    going,
    confirmed_at,
    guests,
    reimbursed_at,
    reimbursement_received_at,
    reimbursement_reference
from game_participants;

drop table game_participants;
alter table game_participants_new rename to game_participants;

create unique index idx_game_participants_game_id_reimbursement_reference
  on game_participants(game_id, reimbursement_reference);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
create table game_participants_new (
    user_id integer not null,
    game_id text not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null,
    going_updated_at datetime default current_timestamp not null,
    going boolean default true,
    confirmed_at datetime default current_timestamp,
    guests integer default 0,
    reimbursed_at datetime,
    reimbursement_received_at datetime,
    reimbursement_reference text,
    primary key (user_id, game_id)
);

insert into game_participants_new(
    rowid,
    user_id,
    game_id,
    created_at,
    updated_at,
    going_updated_at,
    going,
    confirmed_at,
    guests,
    reimbursed_at,
    reimbursement_received_at,
    reimbursement_reference
)
select
    rowid,
    user_id,
    game_id,
    created_at,
    updated_at,
    going_updated_at,
    going,
    confirmed_at,
    guests,
    reimbursed_at,
    reimbursement_received_at,
    reimbursement_reference
from game_participants;

drop table game_participants;
alter table game_participants_new rename to game_participants;

create unique index idx_game_participants_game_id_reimbursement_reference
  on game_participants(game_id, reimbursement_reference);
-- +goose StatementEnd
