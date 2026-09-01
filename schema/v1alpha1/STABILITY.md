# v1.0 stability boundary

The `v1alpha1` spelling is retained to avoid invalidating existing Blueprints,
plans, results, manifests, and capability packs. Beginning with Scaffold Agent
v1.0.0, these five wire contracts are stable for the entire 1.x line.

The release snapshot test pins every schema byte and public API identifier.
Compatible optional additions require an explicit snapshot update and a
compatibility test. Removing a field, changing existing field meaning, making
an optional field required, narrowing an accepted value, or changing a wire
identifier requires a new API version and a Scaffold Agent major release.
