## Summary

<!-- What does this change and why? Link the issue: Closes #123 -->

## Type

- [ ] feat
- [ ] fix
- [ ] refactor / chore
- [ ] docs
- [ ] ci / build

## Domain and contract impact

- [ ] Uses the vocabulary in [CONTEXT.md](../CONTEXT.md) (no new synonyms for existing terms)
- [ ] Changes domain language → CONTEXT.md updated
- [ ] Makes a decision with lasting trade-offs → new ADR in `docs/adr/`
- [ ] Changes the REST API → `api/openapi.yaml` edited first and both sides regenerated (`make generate`)
- [ ] Changes the DB schema → new migration (never edit an applied one)
- [ ] Changes the course graph → `make graph-check` passes and `graph.md` is re-rendered
- [ ] None of the above

## Tests

<!-- How was this verified? Which tests were added/updated? -->

- [ ] Unit tests (domain / components)
- [ ] Integration tests (store / app against Postgres)
- [ ] E2E journey added/updated (if a user-visible flow changed)
- [ ] Tested manually: <!-- steps -->

## Screenshots

<!-- UI changes: before / after. Delete if not applicable. -->

## Checklist

- [ ] PR title follows Conventional Commits (`feat(web): ...`)
- [ ] No secrets, tokens or real learner data committed
- [ ] Answer keys/Explanations are not exposed before a Learner answers
