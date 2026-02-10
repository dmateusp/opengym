-- +goose Up
-- +goose StatementBegin
alter table games drop column waitlist_spots_left;
alter table games drop column max_waitlist_size;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table games add column waitlist_spots_left integer default 0 not null;
alter table games add column max_waitlist_size integer default 0 not null;
-- +goose StatementEnd
