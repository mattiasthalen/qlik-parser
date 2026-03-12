# CLAUDE.md

## Memory

Project-specific memory (conventions, feedback, context) lives in this file (`CLAUDE.md` at the repo root). Do not use `.claude/MEMORY.md` or any other location.

## Conventions

- Commit messages use conventional commits style: `feat:`, `fix:`, `chore:`, `docs:`, etc.
- Merge commits also follows the conventional commit style.
- All commits must be pushed to origin after committing.

## Feedback

- Do not commit failing tests to the branch. Write the test and implementation together, committing only when the test passes. The "red" phase is local-only — never push a failing test as a standalone commit.
- Do not commit to main, it's protected. Always use a worktree.
