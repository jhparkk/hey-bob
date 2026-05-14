# hey-bob
Add configuration and rules for the team management agent

**hey-bob** is a comprehensive framework and project designed to organize the skills and configurations required to operate an autonomous AI agent team. It defines the rules, roles, and collaborative workflows for various agents (such as managers, developers, researchers, and executors) working together to execute tasks effectively.

## Project Structure

- **`workspace/`**: Core environment and configuration for the AI agent team.
  - `AGENTS.md`, `BOOTSTRAP.md`, `SOUL.md`: Global definitions, principles, and initialization details for the agents.
  - `team/`: Contains directories for specific agent roles (`manager/`, `developer/`, `executor/`, `researcher/`), along with team-wide guidelines including `TEAM.md`, `PLAYBOOK.md`, and `ESCALATION.md`.
- **`agent-devs/`**: The primary development workspace where code is written and projects are built. Agents create subdirectories per project (e.g., `/agent-devs/<project_name>/`) to work on technical tasks.
- **`agent-docs/`**: The centralized knowledge base and documentation storage. This serves as the historical record for agent logs, project-specific rules (e.g., `/agent-docs/<project_name>/rules/`), research findings, and reports.

---

## 1. Development Area

| Item | Content |
|------|---------|
| Path | `/agent-devs/<project_name>` |
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