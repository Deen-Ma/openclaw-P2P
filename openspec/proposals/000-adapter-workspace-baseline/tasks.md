# Tasks

1. Import the shared `adapter/` source, tests, manifests, and support scripts
   into `main` without proposal-specific fixes from `001` or `002`.
2. Import the shared `workspace-templates/` files into `main` without proposal-
   specific session-key behavior changes.
3. Add the minimum `.gitignore` rules needed to keep generated `adapter/dist/`
   output and `adapter/node_modules/` out of the tracked baseline.
4. Verify the baseline passes `cd adapter && npm test`.
5. Reconfirm that the rebuilt `001` and `002` proposals will only contain
   proposal-specific deltas.
