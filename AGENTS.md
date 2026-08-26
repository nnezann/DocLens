# AGENTS.md — Rules for Copilot and any other coding agent working on DocLens

This file is binding for every AI coding agent operating in this repository
(GitHub Copilot, Copilot Workspace, Copilot coding agent, or any other
autonomous/semi-autonomous agent). If an instruction here conflicts with a
prompt given in an issue, PR description, or chat, **this file wins** unless
a human explicitly overrides it in writing in that same thread.

---

## 1. Source of truth

Before writing any code, the agent must read, in this order:

1. `DocLens_Engineering_Specification.md` — the system-wide architecture,
   service catalog, event contracts, ownership matrix.
2. The service-specific build prompt for whatever it's building (e.g.
   `DocLens_Identity_Gateway_Build_Prompt.md` for Identity Service / API
   Gateway). Every service should have one of these before an agent starts
   building it — if one doesn't exist yet for the service being requested,
   the agent must stop and ask for it rather than inventing the spec itself.
3. This `AGENTS.md`.

The agent must not deviate from the data models, API contracts, event
names/payloads, or service boundaries defined in those documents without
explicit sign-off from a human in the task thread. If something in a task
request contradicts the spec, the agent flags the conflict and asks rather
than silently picking one.

---

## 2. Technology choices are not the agent's to make

The agent must **never** introduce, swap, or upgrade a technology choice —
a language, framework, database, message broker, queue, cache, cloud
service, third-party API/SDK, or major library — without first stopping and
asking the repository owner. This includes but is not limited to:

- Choosing between alternatives already implied by the spec but not pinned
  (e.g. the spec says "RabbitMQ / Kafka" for service events — the agent must
  ask which one to use before writing any producer/consumer code, and must
  not treat the two as interchangeable or default to whichever is easier to
  scaffold).
- Adding a new dependency that isn't already used elsewhere in the repo for
  the same purpose (e.g. a second HTTP client library, a second ORM, a new
  testing framework).
- Picking a specific managed service (e.g. which email-delivery provider,
  which vector database, which cloud provider) where the spec only names
  the category.
- Upgrading a major version of an existing dependency if it changes public
  behavior.

**What "asking" looks like:** the agent opens a draft PR or posts a comment
(whichever the task's workflow supports) naming the decision, the options,
and a recommendation — but does not merge or continue building on top of
that decision until the human responds. If blocked mid-task by a needed
tech decision, the agent commits what it has so far on the correct branch,
leaves the decision point clearly marked (e.g. a `TODO(decision):` comment
and a note in the PR), and stops rather than guessing and continuing.

Choices that are already explicit and unambiguous in the spec (e.g. "Go" for
the Gateway, "PostgreSQL" for a service that the ownership matrix marks as
PostgreSQL, "JWT" for auth tokens) are pre-approved and don't need to be
re-asked.

---

## 3. Branching model

The repository has four long-lived branches, one per domain:

- `backend`
- `frontend`
- `ai`
- `devops`

Rules:

- The agent identifies which domain a task belongs to **before** writing
  any code, and branches off / commits to a short-lived feature branch cut
  from the matching domain branch (e.g. `backend/identity-service-signup`
  cut from `backend`), not from `main` and not from an unrelated domain
  branch.
- A task that spans domains (e.g. an API contract change that affects both
  `backend` and `frontend`) is split into separate branches/PRs per domain
  — one PR into `backend`, a separate PR into `frontend` — never one PR that
  merges code into two domain branches at once.
- The agent never pushes directly to `backend`, `frontend`, `ai`, or
  `devops` — always through a PR from a feature branch, so there's a review
  point before domain branches move.
- The agent never pushes to or opens a PR against a domain branch that
  doesn't match the work it did (e.g. no infrastructure/CI changes land via
  a `backend/*` branch — those go through `devops`).

---

## 4. Commits and pushes must be atomic

- One logical change per commit. If a change touches multiple unrelated
  concerns (e.g. adding an endpoint *and* fixing an unrelated lint issue),
  split it into separate commits.
- Every commit message uses a short header (imperative mood, ~50 chars,
  scoped to the component) plus a brief description of what and why. Format:

  ```
  <scope>: <short imperative header>

  <1-3 sentence description of what changed and why. Reference the
  spec section or build-prompt step this implements, e.g. "Implements
  Identity Service 2.2.A steps 1-2 (work email capture + code send)".>
  ```

  Example:

  ```
  identity: add signup email verification endpoint

  Implements POST /identity/signup and /identity/signup/verify-email
  per DocLens_Identity_Gateway_Build_Prompt.md section 2.4. Codes
  expire after 10 minutes and allow 5 attempts before requiring a
  resend, per section 2.7.
  ```

- No commit should leave the branch in a broken/non-building state if it
  can reasonably be avoided — prefer several small, individually-working
  commits over one large one, but never a commit that's known to break the
  build "to be fixed in the next commit."
- Push frequently, in small increments, rather than batching a large amount
  of work into one push at the end of a task — this keeps review small and
  makes it easy to spot where a tech-stack decision point was hit.
- Force-pushes to shared feature branches are avoided unless necessary
  (e.g. rebasing before merge); never force-push to a domain branch.

---

## 5. Definition of done for any agent task

A task is not complete until:

1. It matches the relevant spec document(s) — no undocumented deviation.
2. No new technology was introduced without explicit human sign-off (or the
   task is left in a clearly-marked blocked state pending that sign-off).
3. Work is on a feature branch cut from the correct domain branch, opened
   as a PR into that domain branch, not merged directly.
4. Commit history is atomic and each commit follows the header/description
   format above.
5. Tests relevant to the change are included or updated.

## 6. A folder per service.

A task concerns developed a service one by one and therefore one folder will
be used per service with naming conventions of the already existing folders 
in this repo.

## 7. Go is not the language of all services.

Whenever you are developing a service first of all research and analyse the 
best language for the service or component, then document the findings in the
README file per service/folder

## 8. Each service has its own documentation file

Each service you build must have a documentation file explaining its usage and
tech stack used and why.