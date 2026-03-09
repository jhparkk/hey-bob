# PLAYBOOK.md — Team Operations Guide

## Agent Spawn Method

Manager (Bob) uses `sessions_spawn` to spawn a sub-agent.

### Basic Spawn Pattern

```
sessions_spawn(
  task: "<SOUL.md contents> + specific task instruction",
  label: "researcher | developer | executor",
  mode: "run",        # one-time task
  runtime: "subagent"
)
```

### Continued Session Needed (Long Task)

```
sessions_spawn(
  task: "<SOUL.md contents>",
  label: "developer-session",
  mode: "session",    # continuous session
  runtime: "subagent"
)
→ sessions_send(label: "developer-session", message: "Task details")
```

---

## How to Add an Agent

### 1. Create Directory and Files

```bash
mkdir -p team/<new-agent-name>
```

Required File:
- `team/<new-agent-name>/SOUL.md` — Identity, role, behavioral principles

### 2. SOUL.md Write Template

```markdown
# SOUL.md — <Agent Name>

## Identity
- Name: <Name>
- Role: <One-line description of the role>
- Reports to: Manager (Bob)

## Responsibilities
- <Main responsibility 1>
- <Main responsibility 2>

## Behavioral Principles
- Always report results to Manager
- When uncertain, do not guess but ask clearly
- Upon completion, clarify summary + output

## Output Format
Report in the following format upon task completion:
---
✅ Complete: <Task Name>
📋 Result: <Summary>
📁 Deliverables: <File/Link/Data>
⚠️ Issue: <Note if any>
---
```

### 3. Register in TEAM.md Registry

Add a row to the agent registry table in `team/TEAM.md`.

---

## Escalation System

Detailed Protocol → See `team/ESCALATION.md`  
Team Status Board → `team/STATUS.md`

### Summary Flow
```
Agent error occurs
    │
    ├─ Self-healing attempts 1st~3rd
    │       └─ Success → Continue
    │
    └─ 3 failures → Register 🔴 BLOCKED in STATUS.md → Stop
                      │
                      ▼
             Manager checks STATUS.md
                      │
                      ▼
             Escalation report to jhparkk
                      │
                      ▼
             jhparkk direct intervention → Resolution
                      │
                      ▼
             STATUS.md → ✅ RESOLVED
             Report return to Manager → Resume next steps
```

---

## Response to Sub-Agent Timeout

If a sub-agent stops due to a timeout, Manager (Bob) will directly perform the following procedures.

### Verification Steps
1. Verify the existence of actual task deliverables (files, build results, etc.)
2. Verify if the task was practically completed (build, execution, curl tests, etc.)
3. Manager directly supplements missing MD record files

### Principles for Supplementing MD Records
- Verify actual deliverables (code, logs, curl results) directly and write based on facts
- Do not record speculative content
- When supplementing, note `※ Supplemented by Manager post-execution` at the bottom of the file

### runTimeoutSeconds Adjustment Guidelines
| Task Type | Recommended Timeout |
|-----------|---------------------|
| Research (Web search) | 120s |
| Development (Simple code) | 300s |
| Development (Complex project) | 600s |
| Deployment/Execution | 180s |

---

## Current Agent Soul File Locations

| Agent       | File                          |
|-------------|-------------------------------|
| manager     | team/manager/SOUL.md          |
| researcher  | team/researcher/SOUL.md       |
| developer   | team/developer/SOUL.md        |
| executor    | team/executor/SOUL.md         |

---

## Manager (Bob)'s Task Distribution Principles

1. **Researcher First** — Investigate unknowns before proceeding
2. **Developer Based on Specs** — clearly convey research results + requirements
3. **Executor After Verification** — Deploy only what the Developer has finished testing
4. **Parallelize What's Possible** — Spawn independent tasks simultaneously
