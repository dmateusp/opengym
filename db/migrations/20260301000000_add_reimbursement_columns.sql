-- +goose Up
-- +goose StatementBegin
alter table game_participants add column reimbursed_at datetime;
alter table game_participants add column reimbursement_received_at datetime;
alter table game_participants add column reimbursement_reference text;

-- Create a unique constraint on game_id + reimbursement_reference
create unique index idx_game_participants_game_id_reimbursement_reference 
  on game_participants(game_id, reimbursement_reference);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index idx_game_participants_game_id_reimbursement_reference;
alter table game_participants drop column reimbursement_reference;
alter table game_participants drop column reimbursement_received_at;
alter table game_participants drop column reimbursed_at;
-- +goose StatementEnd
