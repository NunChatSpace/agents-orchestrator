-- +goose Up
-- +goose StatementBegin

INSERT INTO worker_groups (name) VALUES
    ('fi-backend'),
    ('fi-frontend'),
    ('ib-kha');

-- Workers with placeholder callback_url and api_key_hash.
-- api_key_hash is SHA-256 of the literal string "changeme-<worker_name>".
-- Replace callback_url and api_key_hash per environment before going live.
INSERT INTO workers (group_id, name, callback_url, api_key_hash, status)
SELECT g.group_id,
       'fi-backend1',
       'http://fi-backend1:9000/jobs',
       'placeholder-hash-fi-backend1',
       'offline'
FROM worker_groups g WHERE g.name = 'fi-backend';

INSERT INTO workers (group_id, name, callback_url, api_key_hash, status)
SELECT g.group_id,
       'fi-backend2',
       'http://fi-backend2:9000/jobs',
       'placeholder-hash-fi-backend2',
       'offline'
FROM worker_groups g WHERE g.name = 'fi-backend';

INSERT INTO workers (group_id, name, callback_url, api_key_hash, status)
SELECT g.group_id,
       'fi-frontend1',
       'http://fi-frontend1:9000/jobs',
       'placeholder-hash-fi-frontend1',
       'offline'
FROM worker_groups g WHERE g.name = 'fi-frontend';

INSERT INTO workers (group_id, name, callback_url, api_key_hash, status)
SELECT g.group_id,
       'fi-frontend2',
       'http://fi-frontend2:9000/jobs',
       'placeholder-hash-fi-frontend2',
       'offline'
FROM worker_groups g WHERE g.name = 'fi-frontend';

INSERT INTO workers (group_id, name, callback_url, api_key_hash, status)
SELECT g.group_id,
       'ib-kha',
       'http://ib-kha:9000/jobs',
       'placeholder-hash-ib-kha',
       'offline'
FROM worker_groups g WHERE g.name = 'ib-kha';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM workers WHERE name IN ('fi-backend1','fi-backend2','fi-frontend1','fi-frontend2','ib-kha');
DELETE FROM worker_groups WHERE name IN ('fi-backend','fi-frontend','ib-kha');
-- +goose StatementEnd
