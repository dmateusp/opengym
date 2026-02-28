-- +goose Up
-- +goose StatementBegin
alter table games
add column locked_at datetime;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table games
drop column locked_at;
-- +goose StatementEnd
