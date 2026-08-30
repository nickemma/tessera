create extension if not exists pgcrypto;

create table if not exists tenants (
    id         uuid primary key default gen_random_uuid(),
    name       text not null,
    status     text not null default 'active',
    created_at timestamptz not null default now()
);

create table if not exists api_keys (
    id           uuid primary key default gen_random_uuid(),
    tenant_id    uuid not null references tenants(id) on delete cascade,
    prefix       text not null,
    key_hash     bytea not null,
    label        text not null default '',
    created_at   timestamptz not null default now(),
    last_used_at timestamptz,
    revoked_at   timestamptz
);

create unique index if not exists api_keys_hash_idx on api_keys (key_hash);
create index if not exists api_keys_tenant_idx on api_keys (tenant_id);

create table if not exists usage_events (
    id             text primary key,
    tenant_id      uuid not null references tenants(id) on delete cascade,
    model          text not null,
    input_tokens   integer not null,
    output_tokens  integer not null,
    cost_usd       numeric(18,8) not null default 0,
    cache_hit      boolean not null default false,
    fallback       boolean not null default false,
    status         text not null,
    occurred_at    timestamptz not null
);

create index if not exists usage_events_tenant_time_idx on usage_events (tenant_id, occurred_at);
