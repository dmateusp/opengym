-- +goose Up
-- +goose StatementBegin
create table games (
  id text primary key, -- case sensitive alphanumeric, 4 characters give 14,776,336 possible values (62^4)
  organizer_id integer not null,
  name text not null,
  description text,
  published_at datetime, -- before that time, the game is only visible to the organizer
  total_price_cents integer default 0 not null, -- total price in cents
  location text, -- location of the game
  starts_at datetime, -- when it starts
  duration_minutes integer default 60 not null, -- total duration in minutes
  max_players integer default -1 not null, -- max number of players that can join the game, -1 = unlimited
  max_waitlist_size integer default 0 not null, -- max number of players that can be on the waitlist, 0 = disabled waitlist, -1 = unlimited
  max_guests_per_player integer default -1 not null, -- max number of guests a single player can bring along, 0 = disabled, -1 = unlimited
  created_at datetime default current_timestamp not null,
  updated_at datetime default current_timestamp not null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table games;
-- +goose StatementEnd
