-- +goose Up
-- +goose StatementBegin
create table game_participants_new (
    id integer primary key autoincrement,
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
    unique (user_id, game_id)
);

-- Keep existing participant insertion order stable by preserving the old rowid as id.
insert into game_participants_new (
    id,
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
from game_participants
order by rowid;

drop table game_participants;
alter table game_participants_new rename to game_participants;

create unique index idx_game_participants_game_id_reimbursement_reference
  on game_participants(game_id, reimbursement_reference);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
create table game_participants_old (
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

insert into game_participants_old (
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
from game_participants
order by id;

drop table game_participants;
alter table game_participants_old rename to game_participants;

create unique index idx_game_participants_game_id_reimbursement_reference
  on game_participants(game_id, reimbursement_reference);
-- +goose StatementEnd
