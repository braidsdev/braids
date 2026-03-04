# Braids

Config-driven API composition — Terraform for APIs.

Declare integrations and custom schemas in a single YAML file, get a unified API. Ships as a single Go binary.

## Install

```sh
# Quick install
curl -sf https://install.braids.dev | sh

# Homebrew
brew install braidsdev/tap/braids

# From source
go install github.com/braidsdev/braids/cmd/braids@latest
```

## Quickstart

```sh
# Scaffold a new project (--demo for a working example with free test APIs)
braids init --demo

# Validate your config
braids validate

# Start the gateway
braids serve
```

## How it works

Define your integrations in `braids.yaml`:

```yaml
version: "1"

connectors:
  stripe:
    type: stripe
    config:
      api_key: ${STRIPE_API_KEY}
  shopify:
    type: shopify
    config:
      shop: ${SHOPIFY_SHOP}
      token: ${SHOPIFY_TOKEN}

schemas:
  customer:
    merge_on: email
    conflict_resolution: prefer_latest
    fields:
      id:
        type: string
      email:
        type: string
      name:
        type: string
      created_at:
        type: datetime

endpoints:
  /customers:
    schema: customer
    sources:
      - connector: stripe
        resource: customers
        mapping:
          id: "'stripe_' + id"
          email: email
          name: name
          created_at: created
      - connector: shopify
        resource: customers
        mapping:
          id: "'shopify_' + id"
          email: email
          name: first_name + ' ' + last_name
          created_at: created_at

server:
  port: 8080
  hot_reload: true
```

Braids fetches upstream APIs in parallel, maps fields into your schema, merges on key, and serves the result.

## CLI Reference

| Command | Description |
|---|---|
| `braids init` | Scaffold a new `braids.yaml` (`--demo` for a working example) |
| `braids validate` | Validate your config file |
| `braids serve` | Start the API gateway |
| `braids version` | Print version info |

Global flag: `-c, --config <path>` — path to config file (default: `braids.yaml`)

## License

[MIT](LICENSE)
