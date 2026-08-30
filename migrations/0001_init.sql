-- +goose Up
create extension if not exists pgcrypto;

create table tenants (
    id         uuid primary key default gen_random_uuid(),
    name       text not null,
    status     text not null default 'active',
    created_at timestamptz not null default now()
);

create table api_keys (
    id           uuid primary key default gen_random_uuid(),
    tenant_id    uuid not null references tenants(id) on delete cascade,
    prefix       text not null,
    key_hash     bytea not null,
    label        text not null default '',
    created_at   timestamptz not null default now(),
    last_used_at timestamptz,
    revoked_at   timestamptz
);

create unique index api_keys_hash_idx on api_keys (key_hash);
create index api_keys_tenant_idx on api_keys (tenant_id);

-- +goose Down
drop table api_keys;
drop table tenants;
