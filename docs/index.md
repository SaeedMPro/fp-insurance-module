---
slug: /
title: Overview
---

# Supplementary Insurance Module

A web application for running an organisation's **supplementary (top-up) health
insurance**: employees submit expense claims, reviewers decide them, and the
system works out what is payable from benefit rules that live in the database
rather than in code.

Built as a bachelor's capstone project at Bu-Ali Sina University. The interface
is entirely Persian and right-to-left; this documentation is in English.

## The problem it addresses

Handling reimbursement by hand fails in three specific ways.

| | Done by hand | With this system |
| --- | --- | --- |
| **Working out the payable amount** | Someone applies a percentage, then a per-claim cap, then checks how much annual allowance is left — per claim, from a spreadsheet. Arithmetic slips, and the employee is the one who notices. | Computed from the rule that was in force on the receipt date. One rounding decision in the whole path. |
| **Answering "why was I paid this?"** | Reconstructed from email and memory. Often unanswerable months later. | Every state change is written to an audit trail in the same transaction as the change, with actor, before and after. |
| **Changing a benefit** | A policy decision waits on whoever maintains the spreadsheet or the code. | An administrator publishes a new rule version through a form. No deployment. |

## The single idea

**Benefit policy is data, not code.** Coverage percentage, per-claim cap, annual
cap, waiting period and eligible family relations are rows in a table, carrying
effective dates. Raising optometry cover from 60% to 80% is a form submission —
and every claim already decided keeps the rule it was priced under.

Everything else in the design exists to keep that property true. If you read one
page beyond this, read [The one idea](start/the-one-idea).

## Who uses it

| Role | What they do |
| --- | --- |
| **Employee** | Submits claims for themselves or a dependent, attaches invoices, watches their remaining allowance. |
| **Reviewer** | Works a queue: approve, reject with a reason, or return a claim for missing documents. |
| **Administrator** | Manages employees and accounts, and publishes coverage-rule versions. |
| **Auditor** | Read-only oversight: the full audit trail and management reports. |
| **Parent HR system** | Syncs the employee roster over an API key. No user account, no interface. |

## Shape of the system

| | |
| --- | --- |
| Backend | Go 1.26, chi, PostgreSQL. 7 service packages over a dependency-free domain. |
| Frontend | React 19, TypeScript, Vite. 17 screens, Persian RTL, Jalali dates. |
| API | 32 paths, 42 operations, described by an OpenAPI document that two tests enforce. |
| Database | 12 tables. Schema is reviewable SQL, not ORM inference. |
| Tests | 54 Go tests across unit, integration and contract layers, plus a 14-step browser suite. |

## Where to go next

- **Evaluating it?** [The one idea](start/the-one-idea), then the [screenshot tour](start/tour).
- **Integrating with it?** [API reference](/api), [Authentication](reference/authentication), [HR integration](reference/hr-integration).
- **Changing it?** [Architecture](engineering/architecture), [Design decisions](engineering/decisions), [Adding a feature](engineering/adding-a-feature).
