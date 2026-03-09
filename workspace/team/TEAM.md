# TEAM.md — Agent Team Manifest

## Team Structure

```
jhparkk (Human)
    │
    ▼
[manager] Bob — Team Operations Manager / Planner
    ├── [researcher] — External Information Research
    ├── [developer]  — Development Design / Coding / Testing
    └── [executor]   — Deployment / Execution
```

## Agent Registry

| ID           | Role                        | Soul File                            | Status     |
|--------------|-----------------------------|--------------------------------------|------------|
| manager      | Team Manager/Planner        | team/manager/SOUL.md                 | ✅ Active  |
| researcher   | Information Research        | team/researcher/SOUL.md              | ✅ Active  |
| developer    | Development/Test            | team/developer/SOUL.md               | ✅ Active  |
| executor     | Deployment/Execution        | team/executor/SOUL.md                | ✅ Active  |

## Communication Protocol

1. **jhparkk → Manager (Bob):** Delivers requirements
2. **Manager → Selected Agent:** Creates sub-agent via `sessions_spawn` and delegates task
3. **Agent → Manager:** Reports results upon completion (sessions_send)
4. **Manager → jhparkk:** Final result report

## Task Flow Example

```
Requirement Received (Manager)
    │
    ├─► researcher: "Research technology X"
    │       └─► Result returned → Manager
    │
    ├─► developer: "Implement Y based on research results"
    │       └─► Code + test results returned → Manager
    │
    └─► executor: "Deploy completed code"
            └─► Deployment results returned → Manager
```

## Team Shared Working Rules
→ Working path/filename: **`team/WORKSPACE.md`**
→ Report/Escalation: **`team/ESCALATION.md`**
→ Team Status Board: **`team/STATUS.md`** (Directly recorded by agent if BLOCKED occurs)

## Team Shared Paths

| Purpose | Path |
|---------|------|
| Code Development Workspace | `/devel/<project_name>` |
| Research/Doc Repository (Obsidian) | `/agent-docs/<project_name>` |

## How to Add an Agent

→ See `team/PLAYBOOK.md`
