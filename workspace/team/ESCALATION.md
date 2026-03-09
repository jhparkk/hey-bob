# ESCALATION.md — Reporting and Escalation System

> All agents must follow this protocol.

---

## Basic Principles

> **"When peaceful, report only to Manager; when on fire, report directly!"**

| Situation | Communication Target |
|-----------|----------------------|
| Normal Progress | Report only to Manager (Bob) |
| Error Occurred | 3 self-healing attempts → Register BLOCKED |
| Decision Needed | Register BLOCKED → Manager escalates to jhparkk |
| Direct Intervention by jhparkk | Communicate directly with the agent → Report return to Manager after resolution |

---

## Agent Self-Healing Rules

In case of an error, proceed as follows:

```
1st attempt: Analyze error message → Identify cause → Modify and retry
2nd attempt: Try a different approach (check dependencies, verify paths, etc.)
3rd attempt: Isolate to a minimal reproduction case and retry

All 3 attempts failed → Register BLOCKED → Stop
```

**Strictly prohibited during self-healing:**
- Executor: Direct modification of source code
- All agents: Unauthorized access to external systems
- All agents: More than 3 repeated attempts (to prevent infinite loops)

---

## How to Register BLOCKED Status

If 3 self-healing attempts fail or a decision from jhparkk is needed:

### 1. Record in STATUS.md
File: `/home/jhpark/hey-bob/workspace/team/STATUS.md`

```markdown
### [BLOCKED] <Role> - <Project Name>
- **Time**: YYYY-MM-DD HH:MM (KST)
- **Agent**: researcher / developer / executor
- **Error/Issue**: (Full error message or situation description)
- **Self Attempts**: 
  - 1st: (Attempt details + result)
  - 2nd: (Attempt details + result)
  - 3rd: (Attempt details + result)
- **Needs**: NEED_USER_INPUT / NEED_MANAGER_DECISION
- **Details**: (What jhparkk or Manager needs to decide)
- **Status**: 🔴 BLOCKED
```

### 2. Stop Task
Do not proceed further after registering BLOCKED. Wait for the next instructions from the Manager.

---

## Manager Monitoring Duties

Manager (Bob) **must always** check STATUS.md upon receiving a sub-agent's result.

```
Sub-agent completion report received
    │
    ├─ No 🔴 BLOCKED in STATUS.md → Proceed to next step
    └─ 🔴 BLOCKED exists in STATUS.md → Escalate to jhparkk
```

### Escalation Message Format to jhparkk
```
🚨 [Escalation] <Project Name>

<Agent> is blocked while working. Intervention by jhparkk is required.

📋 Situation: <Error/Issue Summary>
🔧 Self Attempts: All 3 failed
❓ Decision Needed: <What jhparkk needs to decide>

Would you like to intervene directly?
```

---

## Direct Intervention Procedure

When jhparkk intervenes directly:

1. jhparkk → Direct instruction to the respective agent
2. Agent → Resolves the issue together with jhparkk
3. Resolution Complete → Agent changes BLOCKED to ✅ RESOLVED in STATUS.md
4. jhparkk → Instructs Manager to "Issue resolved, continue"
5. Manager → Resumes next steps

---

## STATUS.md Status Codes

| Code | Meaning |
|------|---------|
| 🔴 BLOCKED | Task stopped, intervention needed |
| 🟡 WAITING | Waiting for results from Manager or other agents |
| 🟢 RESOLVED | Resolution complete (for history preservation) |
