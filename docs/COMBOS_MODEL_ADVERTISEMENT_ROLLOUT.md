# Combo model-list advertisement rollout

Combo advertisement is intentionally disabled by default. Routing and administration remain available, but `/v1/models` and `/models` expose only physical models and existing aliases until the gate is enabled.

## Enable and disable

Set `comboModelAdvertisement` in `config.json` to `true`, restart (or use the normal configuration reload procedure), and verify both model-list aliases. Only fully validated, routable Combos are included. Physical model IDs win on case-insensitive collisions; Combo names retain their configured case and are never duplicated. Listing is read-only and does not reserve round-robin rotation.

Set `comboModelAdvertisement` to `false` and restart to disable advertisement. Existing Combo rows remain in the database and can be re-enabled without recreation. Disabling the advertisement gate does not by itself disable routing; for rollback of runtime behavior, also disable the resolver gate according to the deployment's routing configuration.

A registry/storage failure is fail-open for the physical model list: the endpoint remains successful and omits Combo entries. Invalid candidates, unknown panel models, invalid strategies, and invalid Fusion judges are omitted rather than advertised.

## Migration and rollback

1. Upgrade with the default gate unchanged (`false`).
2. Confirm Combo CRUD, routing, and registry health in staging.
3. Enable the gate for a canary instance or authenticated client cohort.
4. Compare model-list responses from `/v1/models` and `/models`; confirm physical collision behavior and configured casing.
5. Roll back by setting the gate to `false`. If serving errors or latency regress, disable the resolver gate too; retain DB data for later recovery.

## Observability and canary checklist

- Record gate state, registry load success/failure, and count of eligible/omitted Combos without logging prompts or credentials.
- Watch request latency, time-to-first-token, retry/fallback rate, account usage, Fusion quorum failures, and cost separately for Combo traffic.
- Verify no round-robin reservation or rotation revision changes occur during model listing.
- Test direct physical models whose names collide with Combo names; direct routing must win.
- Exercise fallback, round-robin, and Fusion (including invalid judge and unavailable registry) before widening rollout.
- Keep a clear timestamped rollback decision and restore the previous config atomically.
