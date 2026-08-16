# Architecture Decision Records

Short records of the decisions that shaped this codebase — what the situation
was, what was chosen, and what it cost. They exist so the *reasons* survive:
the code shows what was decided, never why, and "why" is what you need before
changing something deliberate.

They are referenced from the code at the points they govern (search for
`ADR-000`) and summarised in [ARCHITECTURE.md](../ARCHITECTURE.md).

| ADR | Decision |
|---|---|
| [0001](0001-layered-architecture.md) | Layered architecture with a dependency-free domain |
| [0002](0002-contract-first-api.md) | The OpenAPI document is the contract, enforced by tests |
| [0003](0003-integer-money.md) | Money is integer rials, with exactly one rounding point |
| [0004](0004-injected-clock-tehran-days.md) | Injected clock, and business days in Asia/Tehran |
