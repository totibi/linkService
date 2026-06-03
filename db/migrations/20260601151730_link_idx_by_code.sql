-- +goose Up
create index if not exists links_by_short_code_idx on links (short_code);


-- +goose Down
drop index links_by_short_code_idx;


