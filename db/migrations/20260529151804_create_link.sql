-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS links (
                                    id  serial,
                                    short_code text not null,
                                    original_url text not null,
                                    created_at TIMESTAMPTZ not null,
                                    visits int
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS links;
-- +goose StatementEnd