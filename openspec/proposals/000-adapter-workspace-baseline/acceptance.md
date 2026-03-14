# Acceptance

## Automated

- `cd adapter && npm test`

## Manual

1. Inspect the baseline import and confirm `adapter/` source, tests, and
   support scripts are tracked.
2. Confirm `workspace-templates/` is tracked.
3. Confirm generated build output and vendored dependencies are not part of the
   tracked baseline.
4. Confirm no `001`-specific detail-handle logic or `002`-specific session
   isolation logic is mixed into the baseline import.

## Commands

```bash
cd adapter && npm test
git diff --stat main...HEAD
git diff --name-only main...HEAD -- adapter workspace-templates .gitignore
```

## Evidence

- baseline branch diff against `main`
- `cd adapter && npm test` output
- tracked file list for `adapter/` and `workspace-templates/`

## Stop Conditions

- stop if the baseline branch still includes `001`-specific or `002`-specific
  behavior changes
- stop if generated `adapter/dist/` or `adapter/node_modules/` content is about
  to be tracked
- stop if the baseline cannot pass `cd adapter && npm test`
