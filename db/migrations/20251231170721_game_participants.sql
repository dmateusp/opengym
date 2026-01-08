-- +goose Up
-- +goose StatementBegin
create table game_participants (
    user_id integer not null,
    game_id text not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null,
    going_updated_at datetime default current_timestamp not null, -- this is the timestamp we use to figure out who makes it into the game and who's in the waitlist
    going boolean default true,
    confirmed_at datetime default current_timestamp, -- this field will be cleared if important details change on the game
    guests integer default 0, -- number of guests the participant is bringing
    primary key (user_id, game_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table game_participants;
-- +goose StatementEnd
