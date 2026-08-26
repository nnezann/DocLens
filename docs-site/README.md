# DocLens documentation portal

This site is the docs-as-code portal for DocLens. It is built with
[Docusaurus](https://docusaurus.io/) and publishes architecture, service
guides, API and event contracts, operations guidance, and ADRs.

## Installation

```bash
npm install
```

## Local Development

```bash
npm run start
```

The local server provides hot reload while documentation is edited.

## Build

```bash
npm run build
```

The production build writes static content to `build/`.

## Contract documentation

Contract source remains with each service:

- OpenAPI documents the public REST gateway.
- Protobuf definitions under each service's `proto/doclens` directory document internal gRPC APIs.
- AsyncAPI definitions document RabbitMQ events as event schemas are formalized.

Buf checks are planned for protobuf formatting, linting, breaking changes, and generated references. Contract checks should run in CI rather than relying on manually copied reference pages.

The deployment target is GitHub Pages using the repository's Docusaurus configuration.
