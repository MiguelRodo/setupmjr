# GitHub Project configuration

| Key | Value |
| --- | --- |
| Contract version | 1 |
| Mode | single |
| Project key | projects |
| Issue repository | MiguelRodo/issues |
| Project owner | MiguelRodo |
| Owner type | user |
| Project number | 40 |
| Project title | projects |
| Routing | label:project:projects |
| Privacy | private issue repository with a public implementation repository |

## Field locations

| Common dimension | Provider location | Provider field |
| --- | --- | --- |
| Class | project field | Class |
| Priority | project field | Priority |
| Status | project field | Status |
| Due date | project field | Target date |
| Parent | native issue relationship | Parent issue |
| Sub-project | repository label | subproject:* |

## Sub-project vocabulary

| Key | Label | Purpose |
| --- | --- | --- |
| setupmjr | subproject:setupmjr | Work owned by MiguelRodo/setupmjr. |

## Routing and grouping

- Required overall Project routing label: `project:projects`.
- setupmjr task issues require `subproject:setupmjr`.
- Project membership and the overall routing label must agree.
- Labels must not duplicate Class, Priority or Status.

## Governance

- Collaboration mode: collaborative administration in a shared Project.
- Issue content lives in the private `MiguelRodo/issues` central issue store; implementation remains in `MiguelRodo/setupmjr`.
- Assignment is explicit only unless a later repository decision says otherwise.
- Exact requested administration requires no separate scope-design source.
