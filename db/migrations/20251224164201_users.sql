-- +goose Up
-- +goose StatementBegin
create table users (
  id integer primary key,
  name text null,
  email text not null unique,
  photo text null,
  created_at datetime default current_timestamp,
  updated_at datetime default current_timestamp
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table users;
-- +goose StatementEnd
