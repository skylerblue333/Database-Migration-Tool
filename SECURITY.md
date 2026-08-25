# Security Policy

Sky Migration Registry is an engineering-beta metadata service. It does not execute SQL.

Do not submit secrets, credentials, tokens, private keys, or sensitive data in migration SQL. The service currently retains registration metadata in process memory and exposes digest/size metadata through its list endpoint.

Current controls include bounded request bodies, strict JSON decoding, safe migration names, conflict detection for reused versions, response security headers, HTTP timeouts, race-tested code, dependency vulnerability scanning, and a non-root container.

Authentication, authorization, TLS termination, persistent encrypted storage, audit-log durability, tenant isolation, database credentials, and production deployment controls are outside the current boundary.
