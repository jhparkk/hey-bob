# REGISTRY.md — Project Registry

> When Bob (Manager) receives a request, he identifies the project here,
> reads the `PROJECT.md` and `rules/<Role>.md` for that project, and passes them to the agent.

---

## Active Projects

| Project Name | Keywords | Status | Path |
|--------------|----------|--------|------|
| test-project | project keywords | 🟢 Active | `test-project/` |

---

## Bob's Project Identification Rules

1. Match **keywords** in the request from jhparkk → Identify the project
2. Once identified, read the `PROJECT.md` of that project
3. Read the corresponding `rules/<Role>.md` for the agent role to be spawned
4. Include the contents of both files in the task to spawn the agent
5. If not identifiable by keywords → Request confirmation from jhparkk: "Which project is this?"

---

## How to Add a Project

1. Create a `./agent-doc/<Project Name>/` folder
2. Write `PROJECT.md` (Overview, Goals, Stack, Path)
3. Write `rules/manager.md`, `rules/researcher.md`, `rules/developer.md`, `rules/executor.md`
4. Add a row to this REGISTRY.md table

---

## Default Path (WSL)

```
/agent-docs/
```
