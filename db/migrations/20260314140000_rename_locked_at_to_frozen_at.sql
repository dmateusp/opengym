-- +goose Up
-- +goose StatementBegin
alter table games rename column locked_at to frozen_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table games rename column frozen_at to locked_at;
-- +goose StatementEnd
