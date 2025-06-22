# Cyclone 🌪️ - AI Code Review Tool

A Go-based tool that integrates with GitHub to provide AI-powered code reviews on pull requests using Claude AI.

## ✨ Features

- **🤖 AI-Powered Reviews**: Uses Claude 3.5 Sonnet for intelligent code analysis
- **📍 Line-Specific Comments**: Comments appear directly on specific lines in the "Files changed" tab
- **📋 Comprehensive Summaries**: Overall PR analysis with structured feedback and poetry
- **🏷️ Categorized Feedback**: Issues tagged by type (nit, suggestion, issue, blocking) and focus area (security, performance, style, etc.)
- **⚙️ Repository-Specific Configuration**: Custom review precision and prompts per repository
- **🔄 Smart Review Triggers**: Only reviews on PR open and ready-for-review events
- **⚡ Real-time Processing**: Responds to PR events via GitHub webhooks
- **🎨 Smart Formatting**: Includes code examples, collaborative language, and lighthearted poems
- **🛡️ Repository Filtering**: Only reviews configured repositories, ignores others

## 🚀 Setup

### 1. Prerequisites
- **Go 1.21+** installed
- **GitHub Personal Access Token** with `repo` and `pull_requests:write` permissions
- **Anthropic API Key** for Claude integration

### 2. Installation
```bash
git clone https://github.com/ThomasPokorny/cyclone-ai.git
cd cyclone-ai
go mod tidy
```

### 3. Configuration
Create a `.env` file in the project root:
```bash
GITHUB_TOKEN=ghp_your_github_token_here
ANTHROPIC_API_KEY=sk-ant-api03-your_anthropic_key_here
PORT=8080
WEBHOOK_SECRET=optional_webhook_secret
```

**Get your API keys:**
- **GitHub Token**: Settings → Developer settings → Personal access tokens
- **Anthropic API Key**: [console.anthropic.com](https://console.anthropic.com) → API Keys

### 4. Create Review Configuration
Create a `review-config.json` file in the project root:
```json
{
  "organizations": [
    {
      "name": "your-github-org",
      "repositories": [
        {
          "name": "critical-service",
          "precision": "strict",
          "custom_prompt": "This is a critical production service. Pay special attention to error handling, performance, and security."
        },
        {
          "name": "frontend-app", 
          "precision": "medium",
          "custom_prompt": "Focus on React best practices, accessibility, and user experience."
        },
        {
          "name": "*",
          "precision": "medium",
          "custom_prompt": "Default configuration for all other repositories."
        }
      ]
    }
  ]
}
```

**Precision levels:**
- `"minor"`: Only critical issues and bugs
- `"medium"`: Balanced review (recommended)
- `"strict"`: Thorough review including style and best practices

### 5. Run Cyclone
```bash
go run main.go
```

### 6. Expose with ngrok (for webhook testing)
```bash
# Install ngrok: https://ngrok.com/download
ngrok http 8080
# Note the https URL (e.g., https://abc123.ngrok.io)
```

### 7. Configure GitHub Webhook
1. Go to your repository → **Settings** → **Webhooks** → **Add webhook**
2. **Payload URL**: `https://your-ngrok-url.ngrok.io/webhook`
3. **Content type**: `application/json`
4. **Events**: Select "Pull requests"
5. **Active**: ✅ Checked
6. Click **Add webhook**

## 🌪️ How It Works

1. **PR Created/Updated** → GitHub sends webhook to Cyclone
2. **Repository Check** → Cyclone verifies if repository is configured for review
3. **Smart Filtering** → Only reviews on `opened` and `ready_for_review` events
4. **Cyclone Fetches** → Gets PR diff and metadata
5. **Claude Analyzes** → AI reviews code using repository-specific configuration
6. **Structured Feedback** → Posts both overall summary and line-specific comments
7. **Categorized Comments** → Each comment tagged by type and priority

## 📝 Review Categories

Cyclone categorizes feedback with emojis and prefixes:

### **Priority Levels:**
- 🧰 **nit**: Minor style/preference issues, non-blocking
- 💡 **suggestion**: Improvements that would be nice but aren't required
- ⚠️ **issue**: Problems that should be addressed before merging
- 🚫 **blocking**: Critical issues that must be fixed
- ❓ **question**: Seeking clarification about intent or approach

### **Focus Areas:**
- 🎨 **style**: Formatting, naming conventions
- ⚡ **perf**: Performance concerns
- 🔒 **security**: Security-related issues
- 📚 **docs**: Documentation needs
- 🧪 **test**: Testing coverage or quality
- 🔧 **refactor**: Code organization improvements

## 🛠️ API Endpoints

- `GET /health` - Health check endpoint
- `POST /webhook` - GitHub webhook receiver
- `GET /` - Basic info about Cyclone

## 🎯 Example Output

**Overall PR Review:**
```
🌪️ Cyclone AI Code Review

✨ Overview
Great work on enhancing the authentication system! This PR brings some solid improvements...

🚀 What's Working Well
- 🔧 Clean dependency injection patterns
- 🛡️ Robust error handling implementation

🎯 Key Areas for Improvement
The JWT token validation could benefit from additional security checks...

---

And now, a little poem about your changes ✨:

*Code reviews with a gentle breeze,*
*Security improvements that aim to please.*
*With tokens checked and errors caught,*
*Quality code is what you've brought!*
```

**Line-Specific Comments:**
```
Line 45 in auth.go:
🌪️ Cyclone: 🔒 security: ⚠️ issue: Consider using bcrypt for password hashing instead of plain text storage

Line 123 in api.js:  
🌪️ Cyclone: 💡 suggestion: 🎨 style: Consider using a more descriptive variable name like 'userCount' instead of 'cnt'
```

## 🔧 Development

### Testing Locally
```bash
# Health check
curl http://localhost:8080/health

# Test webhook (with fake payload)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"action":"opened","pull_request":{"number":123}}'
```

### Project Structure
```
cyclone-ai/
├── cmd/
│   └── cyclone/
│       └── main.go              # Application entry point
├── internal/
│   ├── bot/
│   │   ├── cyclone.go           # Core bot orchestration and setup
│   │   └── webhook.go           # GitHub webhook handling
│   ├── config/
│   │   ├── config.go            # Configuration loading and management
│   │   └── types.go             # Configuration-related types and constants
│   └── review/
│       ├── ai.go                # Claude AI integration and API calls
│       ├── github.go            # GitHub API operations (diff, reviews, comments)
│       ├── parser.go            # Claude response parsing logic
│       └── types.go             # Review-related types and structures
├── .env                         # Environment variables (local development)
├── .gitignore                   # Git ignore rules
├── review-config.json           # Repository review configuration
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
└── README.md                    # This file
```

**Package Responsibilities:**
- **`cmd/cyclone/`** - Application entry point and server startup
- **`internal/bot/`** - Core bot orchestration, HTTP routing, and webhook handling
- **`internal/config/`** - Configuration management (environment variables, JSON config)
- **`internal/review/`** - All review logic (AI integration, GitHub operations, response parsing)
- **`configs/`** - Configuration files for different environments

## ⚡ Next Steps

- [ ] Add support for configuration reloading without restart
- [ ] Implement webhook signature validation for security
- [ ] Create web dashboard for configuration management
- [ ] Add metrics and monitoring capabilities
- [ ] Support for GitHub Apps (beyond Personal Access Tokens)
- [ ] Integration with team coding standards and style guides
- [ ] Multi-organization support with different API keys

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Test with a real PR
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Submit a pull request (and watch Cyclone review it! 🌪️)

**Built with ❤️ by Thomas Pokorny** 🌪️