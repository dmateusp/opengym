-- +goose Up
-- +goose StatementBegin
create table game_participants (
    user_id integer not null,
    game_id text not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null,
    going boolean default true,
    confirmed boolean default true, -- this flag will be set to false if important information changes on the game after the participant has cast his/her vote
    primary key (user_id, game_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table game_participants;
-- +goose StatementEnd
