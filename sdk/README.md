# Mix SDK Generation

Generates TypeScript SDKs using **Speakeasy** from our OpenAPI spec.

## Usage

```bash
make generate-sdk
```

Generates SDK to: `../mix-typescript-sdk/`

## Prerequisites

1. **Install Speakeasy CLI**:
   ```bash
   brew install speakeasy-api/tap/speakeasy
   # or
   curl -fsSL https://go.speakeasy.com/cli-install.sh | sh
   ```

2. **Start dev server**:
   ```bash
   make dev
   ```

3. **Authenticate** (first time):
   ```bash
   speakeasy auth login
   ```

## Files

- `gen.yaml` - Speakeasy configuration
- Generated SDK includes full TypeScript types, examples, and documentation

## Troubleshooting

- **Generation fails**: Ensure `make dev` is running
- **Auth errors**: Run `speakeasy auth login`