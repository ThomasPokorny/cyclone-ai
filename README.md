# Cyclone - AI Code Review Tool

A Go-based tool that integrates with GitHub to provide AI-powered code reviews on pull requests.

## Setup

1. **Install Go** (1.21 or later)

2. **Clone and setup the project**:
   ```bash
   git clone <your-repo>
   cd cyclone
   go mod tidy
   ```

3. **Create a GitHub Personal Access Token**:
    - Go to GitHub Settings > Developer settings > Personal access tokens
    - Generate a new token with these permissions:
        - `repo` (Full control of private repositories)
        - `pull_requests:write` (Write access to pull requests)

4. **Set environment variables**:
   ```bash
   export GITHUB_TOKEN="your_token_here"
   export PORT="8080"
   ```

   Or create a `.env` file (see `.env` example above)

5. **Run the application**:
   ```bash
   go run main.go
   ```

## Testing

Test the health endpoint:
```bash
curl http://localhost:8080/health
```

## Setting up GitHub Webhooks

1. Go to your repository settings
2. Navigate to Webhooks
3. Add webhook with:
    - **Payload URL**: `https://your-domain.com/webhook` (use ngrok for local testing)
    - **Content type**: `application/json`
    - **Events**: Select "Pull requests"

## Next Steps

- [ ] Integrate AI API (Claude, OpenAI, etc.)
- [ ] Add webhook signature validation
- [ ] Improve error handling and logging
- [ ] Add configuration file support
- [ ] Add tests