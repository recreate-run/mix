# Contributing to Mix

Thank you for your interest in contributing to Mix! This document outlines the process for contributing to the project.

## Branch Strategy

Mix follows a structured branching strategy:

- `main`: Production-ready code. Protected branch - no direct pushes allowed.
- `dev`: Development branch where features are integrated before being merged to `main`.
- Feature branches: Created from `dev` for specific features or fixes, typically named with prefixes like `ft-` or `fix-`.

## Pull Request Workflow

### Automated PR from Dev to Main

We have an automated GitHub workflow that creates pull requests from `dev` to `main` whenever changes are pushed to the `dev` branch. This ensures:

1. All code goes through the PR review process before reaching `main`
2. Changes are automatically proposed for merging when they're ready in `dev`

The automated workflow:
- Triggers when code is pushed to `dev`
- Creates a temporary branch (`auto-pr-dev-to-main`) from the current dev state
- Creates a PR from this branch to `main` if there are changes to merge
- Includes a summary of the commits being proposed for merge
- Automatically deletes the temporary branch once the PR is merged or closed

To skip automatic PR creation for a specific commit, include `[skip-auto-pr]` in your commit message.

### Manual PR Process

For contributing features:

1. Create a feature branch from `dev`:
   ```bash
   git checkout dev
   git pull origin dev
   git checkout -b ft-your-feature-name
   ```

2. Make your changes and commit them
3. Push your branch to the repository
4. Create a pull request to merge your feature branch into `dev`
5. After review and approval, merge your PR into `dev`
6. The automated workflow will then create a PR to merge `dev` into `main`

## Development Process

1. Make sure your code passes all tests before submitting a PR
2. Follow the coding guidelines specified in the project
3. Include appropriate documentation for new features
4. Keep PRs focused on a single feature or fix when possible

## Code Review

All pull requests require review before being merged. Reviewers will check for:

- Code quality and adherence to project standards
- Test coverage
- Documentation
- Feature completeness and correctness

## Questions?

If you have questions about contributing, please open an issue in the repository.