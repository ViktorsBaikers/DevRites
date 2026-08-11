# Security review (feature-scoped)

Apply when the feature involves user input, auth/authz, data storage, external
integrations, secrets, or permissions. Delegates to `devrites-audit security`.

## OWASP-Top-10-oriented checks

This is the **single-sourced OWASP-web checklist** both `/rite-review` and the
`devrites-security-auditor` agent apply (trust-boundary + the OWASP **LLM** Top 10 live in
[`standards/security.md`](../../devrites-lib/reference/standards/security.md)). Keep the enumeration here; don't restate it.

- **Injection:** parameterized queries; no string-built SQL/shell/HTML; validate &
  encode at boundaries.
- **Broken access control:** every sensitive action checks authz server-side; no
  trusting client-supplied IDs/roles/tenant; no IDOR, cross-tenant read/write/cache/search,
  confused-deputy path, or privilege change without re-authorization.
- **Auth / session:** secure session handling; no credentials in code/logs; correct
  password/token handling.
- **Sensitive data:** PII/secrets not logged or returned; encryption where required;
  least data exposed.
- **SSRF / external calls:** validate/allowlist outbound URLs; timeouts; don't reflect
  untrusted input into requests.
- **Misconfiguration:** safe defaults; debug off; CORS scoped; security headers as the
  project uses them.
- **Files / request integrity:** resolved paths remain below the allowed root across
  traversal/encoding/symlink/archive cases; state-changing browser requests use the
  framework's forgery protection; CORS is not treated as CSRF defense.
- **Vulnerable dependencies:** new/updated deps audited; no known-vuln versions added.
- **Integrity / deserialization:** safe non-executable parsing with type/size/depth limits;
  untrusted data cannot instantiate executable objects.

## Trust boundary
Apply the three-tier discipline (untrusted → boundary → trusted) per the canonical rule
in [`standards/security.md`](../../devrites-lib/reference/standards/security.md#trust-boundary-three-tiers). A value
that skips the boundary tier is a finding.

## Rules
- Findings labeled Critical/Important/Suggestion. A real auth/data-exposure issue is
  **Critical** → NO-GO until fixed.
- Secrets management: never commit secrets; use the project's secret mechanism.
- Don't expand into a project-wide security audit: feature scope. Record out-of-scope
  risks as follow-ups.
