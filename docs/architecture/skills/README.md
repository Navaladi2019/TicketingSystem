# Architecture squad skills (repo copy)

The live copies of these skills are Multica workspace skills; agents load them from there. This directory is the reviewed source: change a skill here by PR, and after the PR merges the Solution Architect pushes the same text to the workspace with `multica skill update <id> --content-file <path>` (the body only; `name` and `description` are stored as separate fields).

| Skill | Workspace skill id |
|---|---|
| `design-triad` | `d283fa40-9e66-4b8e-a352-e2fa947dbe79` |
| `design-docs` | `cffda1db-eb0c-4ed0-a58f-d587a494efd9` |

If the two copies disagree, the workspace copy is what agents run; open a PR to bring this directory in line.
