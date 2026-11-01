deepmerge 4.3.1 has no upstream release addressing GHSA-gcqx-w48m-w2h4. pnpm applies the patch configured in `pnpm-workspace.yaml` to the source and both distributed JavaScript entrypoints. The frontend and CI pin pnpm 11.22.0 so patch hashes are consistently resolved and installed.

The patch defines copied keys as own data properties, preventing an enumerable `__proto__` key from invoking the inherited setter. Existing unsafe-property guards and ordinary merge semantics remain in place. `src/lib/deepmerge-security.test.ts` exercises the installed Svelte plugin dependency, including poisoned targets, null-prototype targets, nested objects, arrays, symbols, and array replacement.

The dependency is used for local Svelte build configuration. Its version remains 4.3.1, so version-based vulnerability scanners may continue to report the advisory. Remove the patch when a verified upstream fix is available.
