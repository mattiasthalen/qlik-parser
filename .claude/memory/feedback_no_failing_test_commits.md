---
name: no-failing-test-commits
description: Never commit failing tests; only commit tests alongside their passing implementation
type: feedback
---

Do not commit failing tests to the branch. In TDD workflows, write the failing test and the implementation together, committing only when the test passes. The "red" phase is local-only — never push a failing test as a standalone commit.
