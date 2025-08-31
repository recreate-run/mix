# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the development environment.

## Table of Contents

1. [Development Environment Setup Issues](#development-environment-setup-issues)
2. [Backend Service Issues](#backend-service-issues)
3. [Frontend Service Issues](#frontend-service-issues)
4. [Environment Variable Issues](#environment-variable-issues)
5. [Connection Issues](#connection-issues)
6. [Validation Tools](#validation-tools)

## Development Environment Setup Issues

### Problem: `make dev` fails to start services

**Possible causes and solutions:**

1. **Port conflicts**: Check if ports 8088 (backend) or 1420 (frontend) are already in use:
   ```bash
   lsof -i :8088
   lsof -i :1420
   ```
   **Solution**: Kill the existing processes:
   ```bash
   lsof -ti :8088 | xargs kill -9
   lsof -ti :1420 | xargs kill -9
   ```

2. **Missing dependencies**: Run:
   ```bash
   make install-deps
   ```

3. **Service startup errors**: Check logs:
   ```bash
   make tail-log
   ```

### Problem: Missing tools or dependencies

**Solution**: Run the environment validation script:
```bash
make test-env
```

## Backend Service Issues

### Problem: Backend fails to start

**Possible causes and solutions:**

1. **Compilation errors**: Check for Go build errors in the logs:
   ```bash
   make tail-log
   ```

2. **Configuration issues**: Validate your environment:
   ```bash
   make test-env
   ```

3. **Port already in use**: Check if something is using port 8088:
   ```bash
   lsof -i :8088
   ```

### Problem: Backend crashes immediately

**Possible causes and solutions:**

1. **Missing environment variables**: Make sure your `.env` file has all required variables.

2. **Database connection issues**: Check database connection in logs:
   ```bash
   make tail-log
   ```

## Frontend Service Issues

### Problem: Frontend fails to connect to backend

**Possible causes and solutions:**

1. **Missing environment variable**: Ensure `VITE_BACKEND_URL` is set correctly in `tauri_app/.env`:
   ```
   VITE_BACKEND_URL=http://localhost:8088
   ```

2. **Backend not running**: Check if backend is running:
   ```bash
   make test-connection
   ```

3. **Port conflict**: If the backend fails to start due to port conflict:
   ```bash
   lsof -i :8088  # Check what's using the port
   lsof -ti:8088 | xargs kill -9  # Kill the process
   ```

### Problem: Frontend build fails

**Possible causes and solutions:**

1. **Missing dependencies**: Run:
   ```bash
   cd tauri_app && bun i
   ```

2. **Typescript errors**: Check logs for type errors:
   ```bash
   make tail-log
   ```

3. **Frontend not starting**: If you see errors in the frontend startup:
   ```bash
   # Ensure the frontend environment is properly set up
   echo "VITE_BACKEND_URL=http://localhost:8088" > tauri_app/.env
   
   # Check if any processes are blocking the frontend port
   lsof -i :1420
   lsof -ti:1420 | xargs kill -9  # If needed
   ```

## Environment Variable Issues

### Problem: Missing environment variables

**Solution**: Create or update your `.env` file with required variables:
```bash
# Essential variables
POSTHOG_API_KEY=your_key_here
SIDECAR_ENABLED=false
VITE_BACKEND_URL=http://localhost:8088
```

You can also validate your environment with:
```bash
make test-env
```

## Connection Issues

### Problem: Frontend can't connect to backend

**Possible causes and solutions:**

1. **Backend not running**: Make sure backend is running on port 8088.

2. **Incorrect backend URL**: Check `tauri_app/.env` has:
   ```
   VITE_BACKEND_URL=http://localhost:8088
   ```

3. **Network issues**: Try the connection test:
   ```bash
   make test-connection
   ```

## Validation Tools

We've provided several testing and validation tools to help diagnose issues:

### Environment Variable Validation
```bash
make test-env
```
Checks that all required environment variables are set properly.

### Development Environment Validation
```bash
make test-dev-env
```
Validates the overall development environment setup.

### Connection Test
```bash
make test-connection
```
Tests the connection between frontend and backend services.

### Run All Tests
```bash
make test-all
```
Runs all validation tests in sequence.

## Getting Further Help

If you've tried the steps above and are still experiencing issues:

1. Check the detailed logs:
   ```bash
   make tail-log
   ```

2. Create a new issue on the GitHub repository with:
   - Description of the problem
   - Steps to reproduce
   - Output of `make test-all`
   - Relevant sections from the logs