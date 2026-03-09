# WORKSPACE.md — Agent Common Workspace Rules

> ⚠️ All agents must follow the rules in this file.

---

## 1. Development Area

| Item | Content |
|------|---------|
| Path | `/devel/<project_name>` |
| Purpose | Workspace for writing code and development tasks |
| Owner | Developer, Executor |

- Create a folder under `<project_name>` based on the task assigned by jhparkk.
- Example: `/agent-devs/api-server/`

---

## 2. Agent Documentation Area (Obsidian)

| Item | Content |
|------|---------|
| Path | `/agent-docs/<project_name>` |
| Purpose | Storing agent logs, research findings, and reports |
| Owner | All agents (Create files based on roles) |

---

## 3. File Naming Rules

### Default Task File
```
[ProjectName]Role_YYYYMMDD.md
```

### Submission File (Derived tasks from additional instructions or agent interaction)
```
ProjectName_Role_YYYYMMDD_SubmissionName.md
```

### Role Identifiers
| Role | Identifier |
|------|------------|
| Team Manager | `manager` |
| Information Research | `researcher` |
| Development/Test | `developer` |
| Deployment/Execution | `executor` |

### Example
```
api-server_manager_20260306.md
api-server_researcher_20260306.md
api-server_developer_20260306.md
api-server_executor_20260306.md
api-server_developer_20260306_api_modification.md
```

---

## 4. Glossary

- **Task**: Project-level task directly instructed by jhparkk
- **Submission**: Sub-task derived from additional instructions by jhparkk or interactions between agents during task execution
- **Role**: Role of the agent who wrote the file (manager / researcher / developer / executor)

---

## 5. Workflow Summary

```
Instruction from jhparkk (Project name confirmed)
    │
    ▼
[Obsidian] /agent-docs/<project_name>/ folder created
    │
    ├─ [Project]manager_Date.md     ← Planning and task breakdown
    ├─ [Project]researcher_Date.md  ← Research results
    ├─ [Project]developer_Date.md   ← Development records
    └─ [Project]executor_Date.md    ← Deployment records

[Code] /devel/<project_name>/    ← Actual code
```
