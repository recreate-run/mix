# Setting Up Personal Access Token for API Documentation Workflow

The automated API documentation workflow requires a Personal Access Token (PAT) with appropriate permissions to create pull requests. Follow these steps to set it up:

## 1. Create a Personal Access Token (PAT)

1. Go to GitHub Settings -> Developer settings -> Personal access tokens -> Fine-grained tokens
2. Click "Generate new token"
3. Give it a descriptive name like "API Documentation Automation"
4. Set the expiration to a reasonable period (recommended: at least 90 days)
5. Select the specific repository where this workflow will run
6. Grant the following permissions:
   - Repository permissions:
     - Contents: Read and write
     - Pull requests: Read and write
     - Metadata: Read-only (automatically selected)

## 2. Add the Token as a Repository Secret

1. Go to your repository on GitHub
2. Navigate to Settings -> Secrets and variables -> Actions
3. Click "New repository secret"
4. Name: `API_DOCS_PAT`
5. Value: Paste the token you created above
6. Click "Add secret"

## 3. Security Considerations

- This token has permissions to commit to your repository and create pull requests
- It's recommended to use a dedicated GitHub account for automation if possible
- Regularly rotate the token (create a new one and update the secret)
- Grant only the minimum permissions necessary

Once you've set up the PAT as a repository secret, the workflow will be able to create pull requests on your behalf.