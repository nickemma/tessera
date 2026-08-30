# TESSERA v0 Threat Model

## Protected assets

- Tenant identity and API keys.
- Tenant prompts and model responses.
- Token budgets and usage records.

## Trust boundaries

- Client to gateway: untrusted network and credentials.
- Gateway to PostgreSQL, Redis, and model provider: service credentials and network policy.
- Tenant to tenant: no shared identity, budget, or cache state.

## Required controls

- Store API-key hashes, never raw keys.
- Reject revoked keys and inactive tenants.
- Apply tenant context before accessing tenant-owned data.
- Budget checks occur before model dispatch.
- Redis failures fail closed for budget enforcement.
- Do not log prompts, raw keys, or model responses by default.

Production TLS, secret management, and network policies remain deployment responsibilities for the v0 environment.
