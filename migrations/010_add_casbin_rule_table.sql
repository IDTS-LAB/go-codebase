-- +goose Up
CREATE TABLE casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(255) DEFAULT '',
    v1 VARCHAR(255) DEFAULT '',
    v2 VARCHAR(255) DEFAULT '',
    v3 VARCHAR(255) DEFAULT '',
    v4 VARCHAR(255) DEFAULT '',
    v5 VARCHAR(255) DEFAULT ''
);

CREATE INDEX idx_casbin_rule_ptype ON casbin_rule(ptype);

-- +goose Down
DROP TABLE IF EXISTS casbin_rule;
