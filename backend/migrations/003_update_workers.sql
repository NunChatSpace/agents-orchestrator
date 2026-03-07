-- +goose Up
-- +goose StatementBegin

-- Replace placeholder hashes with real SHA-256 hashes of the dev raw keys.
-- Raw keys are: dev-key-<worker-name>  (set AUTH_KEY to the same value in each WAgent container).
-- OAgent sends auth_key = hash to WAgent; WAgent sends X-Worker-Key = raw key back to OAgent.
UPDATE workers SET
    api_key_hash = encode(sha256('dev-key-fi-backend1'::bytea), 'hex'),
    status       = 'idle'
WHERE name = 'fi-backend1';

UPDATE workers SET
    api_key_hash = encode(sha256('dev-key-fi-backend2'::bytea), 'hex'),
    status       = 'idle'
WHERE name = 'fi-backend2';

UPDATE workers SET
    api_key_hash = encode(sha256('dev-key-fi-frontend1'::bytea), 'hex'),
    status       = 'idle'
WHERE name = 'fi-frontend1';

UPDATE workers SET
    api_key_hash = encode(sha256('dev-key-fi-frontend2'::bytea), 'hex'),
    status       = 'idle'
WHERE name = 'fi-frontend2';

UPDATE workers SET
    api_key_hash = encode(sha256('dev-key-ib-kha'::bytea), 'hex'),
    status       = 'idle'
WHERE name = 'ib-kha';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE workers SET
    api_key_hash = 'placeholder-hash-' || name,
    status       = 'offline'
WHERE name IN ('fi-backend1','fi-backend2','fi-frontend1','fi-frontend2','ib-kha');
-- +goose StatementEnd
