# Tech Debt Registry

## Open Items

(none)

## Resolved Items

### TD-001: JSON envelope omitempty deviates from spec

**Registered:** 2026-03-11
**Source:** PLN-2026-001 (safe-ify v1 scaffold), T13 Code Quality Review
**Resolved:** 2026-03-11 — removed omitempty, JSON now renders explicit null

### TD-002: Custom bytesReader in applications.go

**Registered:** 2026-03-11
**Source:** PLN-2026-001 (safe-ify v1 scaffold), T13 Code Quality Review
**Resolved:** 2026-03-11 — replaced with bytes.NewReader from stdlib
