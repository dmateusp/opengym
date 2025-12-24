-- +goose Up
-- +goose StatementBegin
create table users (
  id integer primary key,
  name text,
  email text not null unique,
  photo text,
  created_at datetime default current_timestamp not null,
  updated_at datetime default current_timestamp not null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table users;
-- +goose StatementEnd
