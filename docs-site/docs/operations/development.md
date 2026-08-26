---
title: Development workflow
---

Each service is independently runnable and documented in its own folder. Keep service APIs, configuration, ownership, and operational behavior close to the implementation README, then link the stable summary into this portal.

```bash
cd docs-site
npm install
npm start
```

Build the static site with:

```bash
npm run build
```

Contract checks should run in CI as schemas are introduced: OpenAPI validation, Buf lint and breaking-change checks, and AsyncAPI validation.
