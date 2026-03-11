# Project Memory

## Conventions

- Commit messages use conventional commits style: `feat:`, `fix:`, `chore:`, `docs:`, etc.
- All commits must be pushed to origin after committing.
- Memories are stored in the repo at `.claude/MEMORY.md`, not in the global Claude directory.

## Feedback

- Do not commit failing tests to the branch. Write the test and implementation together, committing only when the test passes. The "red" phase is local-only — never push a failing test as a standalone commit.
