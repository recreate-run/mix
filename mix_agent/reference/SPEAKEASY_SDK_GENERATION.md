# Speakeasy SDK Generation Guide

## Overview

This guide provides comprehensive documentation for generating client SDKs using Speakeasy from our OpenAPI endpoint at `http://localhost:8088/doc`. Speakeasy is a platform that automatically generates type-safe, production-ready SDKs in multiple programming languages from OpenAPI specifications.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Prerequisites](#prerequisites)
3. [Installation & Setup](#installation--setup)
4. [OpenAPI Endpoint Configuration](#openapi-endpoint-configuration)
5. [SDK Generation Process](#sdk-generation-process)
6. [Supported Languages](#supported-languages)
7. [Configuration Files](#configuration-files)
8. [Customization Options](#customization-options)
9. [Testing Generated SDKs](#testing-generated-sdks)
10. [Publishing SDKs](#publishing-sdks)
11. [CI/CD Integration](#cicd-integration)
12. [Best Practices](#best-practices)
13. [Troubleshooting](#troubleshooting)

## Quick Start

```bash
# 1. Install Speakeasy CLI
curl -fsSL https://go.speakeasy.com/cli-install.sh | sh

# 2. Start local development server
make dev

# 3. Initialize Speakeasy project
speakeasy quickstart

# 4. Generate SDK using local OpenAPI endpoint
speakeasy generate sdk --schema http://localhost:8088/doc --lang typescript
```

## Prerequisites

### System Requirements

- **Operating System**: macOS, Linux, or Windows
- **Internet Connection**: Required for CLI authentication and spec validation
- **Git**: For version control and GitHub integration
- **Development Server**: Local server running at `http://localhost:8088`

### Account Requirements

- **Speakeasy Account**: Free account at [https://app.speakeasy.com](https://app.speakeasy.com)
- **Account Tiers**:
  - **Free Trial**: 14-day business tier trial (no credit card required)
  - **Free Tier**: 1 SDK with up to 50 API methods
  - **Business Tier**: Unlimited SDKs and advanced features

### OpenAPI Specification Support

- **Supported Formats**: OpenAPI 3.0, OpenAPI 3.1, JSON Schema
- **Current Implementation**: OpenAPI 3.1 specification served at `http://localhost:8088/doc`
- **Conversion Tools**: Available for Swagger 2.0 and Postman collections

## Installation & Setup

### 1. Install Speakeasy CLI

Choose your preferred installation method:

#### macOS (Homebrew)

```bash
brew install speakeasy-api/tap/speakeasy
```

#### Universal Script

```bash
curl -fsSL https://go.speakeasy.com/cli-install.sh | sh
```

#### Windows (Winget)

```bash
winget install --id=Speakeasy.speakeasy
```

#### Windows (Chocolatey)

```bash
choco install speakeasy
```

### 2. Verify Installation

```bash
speakeasy --version
```

### 3. Authenticate with Speakeasy

```bash
speakeasy quickstart
```

This command opens your browser for workspace authentication.

## OpenAPI Endpoint Configuration

### Current Implementation

Our application serves the OpenAPI specification at:

- **Endpoint**: `http://localhost:8088/doc`
- **Format**: OpenAPI 3.1 JSON specification
- **Source**: Hardcoded specification in `mix_agent/internal/http/rest_docs.go`

### Starting the Development Server

```bash
# Start both frontend and backend with auto-reload
make dev

# Verify the OpenAPI endpoint is accessible
curl http://localhost:8088/doc | jq '.'
```

### OpenAPI Specification Features

Our current specification includes:

- **17+ REST Endpoints**: Complete API coverage
- **Authentication**: Security schemes and authentication methods
- **Request/Response Models**: Comprehensive data type definitions
- **Error Handling**: Standardized error response formats
- **API Versioning**: Version management structure

## SDK Generation Process

### 1. Interactive Generation (Recommended for First Time)

```bash
speakeasy quickstart
```

This interactive process will:

1. Prompt for OpenAPI document URL: `http://localhost:8088/doc`
2. Ask for SDK name (recommended: company/project name)
3. Let you select target language
4. Validate specification and generate SDK
5. Set up basic configuration files

### 2. Direct SDK Generation

```bash
# Generate TypeScript SDK
speakeasy generate sdk --schema http://localhost:8088/doc --lang typescript --out ./sdks/typescript

# Generate Python SDK
speakeasy generate sdk --schema http://localhost:8088/doc --lang python --out ./sdks/python

# Generate Go SDK
speakeasy generate sdk --schema http://localhost:8088/doc --lang go --out ./sdks/go
```

### 3. Using Configuration File

```bash
# Generate using gen.yaml configuration
speakeasy run
```

## Supported Languages

### General Availability (Production Ready)

| Language | Maturity | Features | Package Manager |
|----------|----------|----------|-----------------|
| **TypeScript** | GA | Full feature set | npm/yarn |
| **Python** | GA | Full feature set | pip/poetry |
| **Go** | GA | Full feature set | go modules |
| **Java** | GA | Full feature set | maven/gradle |
| **C#** | GA | Full feature set | NuGet |
| **PHP** | GA | Level 1 support | Composer |

### Beta Languages

| Language | Maturity | Features | Package Manager |
|----------|----------|----------|-----------------|
| **Ruby** | Beta | Level 1 support | RubyGems |
| **Unity** | Beta | Level 1 support | Unity Package Manager |
| **MCP TypeScript** | Beta | Level 2 support | npm/yarn |

### Additional Targets

- **Terraform Providers**: GA maturity, Level 2 support
- **Postman Collections**: Alpha maturity, Level 1 support
- **C++, Swift, Rust**: Available with varying maturity levels

## Configuration Files

### 1. gen.yaml Configuration

Create a `gen.yaml` file in your project root:

```yaml
configVersion: 2.0.0
generation:
  sdkClassName: MixSDK
  maintainOpenAPIOrder: true
  usageSnippets:
    optionalPropertyRendering: withExample
  fixes:
    nameResolutionDec2024: true
    parameterOrderingFeb2024: true
    requestResponseComponentNamesFeb2024: true
management:
  docChecksum: # Generated automatically
  docVersion: 1.0.0
  speakeasyVersion: 1.400.0
  generationVersion: 2.443.5
  releaseVersion: 1.0.0
  configChecksum: # Generated automatically
typescript:
  version: 1.0.0
  packageName: "@your-company/mix-sdk"
  author: "Your Company"
  clientServerStatusCodesAsUnions: true
  enumFormat: union
  flattenGlobalSecurity: true
  imports:
    option: openapi
    paths:
      callbacks: models/callbacks
      errors: models/errors
      operations: models/operations
      shared: models/shared
      webhooks: models/webhooks
  inputModelSuffix: input
  maxMethodParams: 4
  methodArguments: require-security-and-request
  moduleFormat: commonjs
  outputModelSuffix: output
  packageVersion: 1.0.0
  responseFormat: flat
```

### 2. Workflow Configuration

Create `.speakeasy/workflow.yaml`:

```yaml
workflowVersion: 1.0.0
sources:
  my-source:
    inputs:
      - location: http://localhost:8088/doc
targets:
  typescript:
    target: typescript
    source: my-source
  python:
    target: python
    source: my-source
  go:
    target: go
    source: my-source
```

## Customization Options

### 1. SDK Structure Customization

- **Package Names**: Configure package names for different languages
- **Class Names**: Customize generated class and method names
- **Import Paths**: Control module organization and import structure
- **Response Formats**: Choose between flat or nested response structures

### 2. Authentication Customization

- **Security Schemes**: Configure OAuth, API keys, bearer tokens
- **Global Security**: Set default authentication methods
- **Per-Operation Security**: Override security for specific endpoints

### 3. Code Generation Options

- **Enum Formats**: Choose between union types or traditional enums
- **Parameter Handling**: Configure method parameter organization
- **Error Handling**: Customize error response structures
- **Documentation**: Control code comment generation

### 4. Using OpenAPI Overlays

Create overlays to enhance your OpenAPI spec without modifying the source:

```yaml
# overlay.yaml
overlay: 1.0.0
info:
  title: Mix API Enhancements
  version: 1.0.0
actions:
  - target: "$.info"
    update:
      x-speakeasy-name-override: MixAPI
  - target: "$.paths[*][*]"
    update:
      x-speakeasy-usage-example: true
```

Apply overlay during generation:

```bash
speakeasy generate sdk --schema http://localhost:8088/doc --overlay overlay.yaml --lang typescript
```

## Testing Generated SDKs

### 1. Built-in SDK Testing

```bash
# Run SDK tests
speakeasy test

# Run tests with custom configuration
speakeasy test --config test-config.yaml
```

### 2. Custom Test Configuration

Create `test-config.yaml`:

```yaml
tests:
  - name: basic-functionality
    steps:
      - type: http
        method: GET
        url: /api/status
        expect:
          status: 200
      - type: sdk
        language: typescript
        code: |
          const client = new MixSDK({ apiKey: "test" });
          const result = await client.status.get();
          expect(result.status).toBe("ok");
```

### 3. Integration Testing

```bash
# Start local server
make dev

# Run integration tests against local API
speakeasy test --baseURL http://localhost:8088
```

## Publishing SDKs

### 1. Package Manager Publishing

#### TypeScript/npm

```bash
cd sdks/typescript
npm publish
```

#### Python/PyPI

```bash
cd sdks/python
python -m pip install build twine
python -m build
twine upload dist/*
```

#### Go Modules

```bash
cd sdks/go
git tag v1.0.0
git push origin v1.0.0
```

### 2. GitHub Releases

```bash
# Tag and release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Create GitHub release
gh release create v1.0.0 --title "SDK v1.0.0" --notes "Initial SDK release"
```

## CI/CD Integration

### 1. GitHub Actions Workflow

Create `.github/workflows/generate-sdks.yml`:

```yaml
name: Generate SDKs
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  generate-sdks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Speakeasy CLI
        run: |
          curl -fsSL https://go.speakeasy.com/cli-install.sh | sh
          echo "$HOME/.speakeasy/bin" >> $GITHUB_PATH
      
      - name: Start Development Server
        run: |
          make dev &
          sleep 30  # Wait for server to start
      
      - name: Generate SDKs
        run: speakeasy run
        env:
          SPEAKEASY_API_KEY: ${{ secrets.SPEAKEASY_API_KEY }}
      
      - name: Run SDK Tests
        run: speakeasy test
      
      - name: Publish SDKs
        if: github.ref == 'refs/heads/main'
        run: |
          # Add publishing commands here
```

### 2. Automated Versioning

```yaml
      - name: Update Version
        run: |
          VERSION=$(date +%Y%m%d%H%M%S)
          speakeasy configure sources --set-version $VERSION
```

## Best Practices

### 1. OpenAPI Specification Best Practices

- **Use Descriptive Names**: Clear operation IDs and parameter names
- **Include Examples**: Provide request/response examples for better SDK generation
- **Proper Data Types**: Use specific types and formats (e.g., `date-time`, `email`)
- **Error Responses**: Document all possible error responses
- **Security Schemes**: Properly define authentication methods

### 2. SDK Configuration Best Practices

- **Consistent Naming**: Use consistent naming conventions across languages
- **Version Management**: Implement semantic versioning for your SDKs
- **Documentation**: Generate comprehensive documentation with examples
- **Testing**: Implement thorough testing for all generated SDKs

### 3. Development Workflow Best Practices

- **Source Control**: Version control your Speakeasy configuration files
- **Automated Testing**: Set up CI/CD pipelines for SDK generation and testing
- **Documentation**: Keep this documentation updated with changes
- **Monitoring**: Monitor SDK usage and performance

## Troubleshooting

### Common Issues and Solutions

#### 1. OpenAPI Endpoint Not Accessible

```bash
# Check if development server is running
make tail-log

# Verify endpoint accessibility
curl -I http://localhost:8088/doc
```

#### 2. Authentication Issues

```bash
# Re-authenticate with Speakeasy
speakeasy auth logout
speakeasy quickstart
```

#### 3. Invalid OpenAPI Specification

```bash
# Validate OpenAPI spec
speakeasy validate openapi --schema http://localhost:8088/doc

# Check for common issues
speakeasy lint openapi --schema http://localhost:8088/doc
```

#### 4. Generation Failures

```bash
# Check Speakeasy logs
speakeasy generate sdk --schema http://localhost:8088/doc --lang typescript --debug

# Validate configuration
speakeasy validate gen.yaml
```

#### 5. SDK Testing Failures

```bash
# Run tests with verbose output
speakeasy test --verbose

# Check test configuration
speakeasy validate test-config.yaml
```

### Getting Help

1. **Speakeasy Documentation**: [https://docs.speakeasy.com](https://docs.speakeasy.com)
2. **GitHub Issues**: Report issues in the Speakeasy repository
3. **Community Discord**: Join the Speakeasy community for support
4. **Support Email**: Contact Speakeasy support for technical issues

## Advanced Topics

### 1. Custom Code Generation Templates

For advanced customization, you can create custom templates for specific languages or modify the default generation behavior.

### 2. Multiple API Versions

Configure Speakeasy to handle multiple API versions simultaneously:

```yaml
sources:
  v1-api:
    inputs:
      - location: http://localhost:8088/v1/doc
  v2-api:
    inputs:
      - location: http://localhost:8088/v2/doc
```

### 3. SDK Documentation Integration

Integrate generated SDKs with documentation platforms like:

- **Mintlify**: Automatic documentation generation
- **Scalar**: Interactive API documentation
- **GitBook**: Documentation hosting and management

## Conclusion

This guide provides comprehensive coverage of SDK generation using Speakeasy with our OpenAPI endpoint. The generated SDKs are production-ready and provide type-safe, idiomatic client libraries for multiple programming languages.

For the most up-to-date information and advanced features, refer to the official Speakeasy documentation and the extensive documentation files in the `speakeasy_docs/` directory.

---

**Last Updated**: Generated from comprehensive analysis of Speakeasy documentation  
**Version**: 1.0.0  
**Maintainer**: Development Team
