### Tradeoffs decisions

| Chose | Over | Why here | When the other wins |
|--------|------|------|---------------------|
|  internal/ split now     | one main.go, refactor later |  Five minutes now versus a painful untangle in week 2. The split is what makes the service testable without a database.   |  Flat wins for a script, a spike, or anything you'll delete. It genuinely is simpler and you shouldn't feel bad using it.                 |
|  Typed structs + json tags    | map[string]any |  The compiler checks your field names, and the struct is self-documenting — someone reading chat.Request knows the API.  |  Maps win when the shape is genuinely unknown at compile time — a passthrough proxy, a plugin payload. You'll actually use one in M4 for provider-specific options.          |
|   DisallowUnknownFields()    | Silently ignore extras  | A client sending {"promt":"hi"} gets told, instead of getting an empty prompt and wondering why. Typos surface immediately.    |  Lenient wins for public APIs where old clients must keep working as you add fields — Postel's law. Strict wins for internal APIs where a typo should be loud. Ours is a platform API with paying tenants; loud is right.                  |
|   Env config    | YAML file, CLI flags  | One mechanism, works identically on your laptop and in a container, and it's what Kubernetes hands you in M9. | Files win for deeply nested config. Flags win for CLI tools where the user types them.                 |
|   log/slog JSON   | log, zap, zerolog | Stdlib since 1.21, structured, no dependency. Machine-searchable from commit one. | zap or zerolog win when logging is genuinely on your hot path and allocations matter — you'll measure whether that's true in M5 rather than assuming it.            |
