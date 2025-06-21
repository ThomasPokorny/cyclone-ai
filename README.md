# Cyclone 🌪️ - AI Code Review Tool

A Go-based tool that integrates with GitHub to provide AI-powered code reviews on pull requests using Claude AI.

## ✨ Features

- **🤖 AI-Powered Reviews**: Uses Claude Sonnet 4 for intelligent code analysis
- **📍 Line-Specific Comments**: Comments appear directly on specific lines in the "Files changed" tab
- **📋 Comprehensive Summaries**: Overall PR analysis with structured feedback
- **🏷️ Categorized Feedback**: Issues tagged by type (nit, suggestion, issue, blocking) and focus area (security, performance, style, etc.)
- **⚡ Real-time Processing**: Responds to PR events via GitHub webhooks
- **🔄 Concurrent Handling**: Processes multiple PRs simultaneously
- **🎨 Smart Formatting**: Includes code examples, collaborative language, and even lighthearted poems!

## 🚀 Setup

### 1. Prerequisites
- **Go 1.21+** installed
- **GitHub Personal Access Token** with `repo` and `pull_requests:write` permissions
- **Anthropic API Key** for Claude integration

### 2. Installation
```bash
git clone <your-repo>
cd cyclone
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

### 4. Run Cyclone
```bash
go run main.go
```

### 5. Expose with ngrok (for webhook testing)
```bash
# Install ngrok: https://ngrok.com/download
ngrok http 8080
# Note the https URL (e.g., https://abc123.ngrok.io)
```

### 6. Configure GitHub Webhook
1. Go to your repository → **Settings** → **Webhooks** → **Add webhook**
2. **Payload URL**: `https://your-ngrok-url.ngrok.io/webhook`
3. **Content type**: `application/json`
4. **Events**: Select "Pull requests"
5. **Active**: ✅ Checked
6. Click **Add webhook**

## 🌪️ How It Works

1. **PR Created/Updated** → GitHub sends webhook to Cyclone
2. **Cyclone Fetches** → Gets PR diff and metadata
3. **Claude Analyzes** → AI reviews code for quality, security, bugs, style
4. **Structured Feedback** → Posts both overall summary and line-specific comments
5. **Categorized Comments** → Each comment tagged by type and priority

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

Summary: This PR adds user authentication with JWT tokens...
[Detailed analysis]

[Lighthearted poem about the changes]
```

**Line-Specific Comments:**
```
Line 45 in auth.go:
🔒 security: ⚠️ issue: Consider using bcrypt for password hashing instead of plain text storage

Line 123 in api.js:  
💡 suggestion: 🎨 style: Consider using a more descriptive variable name like 'userCount' instead of 'cnt'
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
cyclone/
├── main.go           # Main application
├── go.mod           # Go module definition
├── .env             # Environment variables (local)
├── .gitignore       # Git ignore rules
└── README.md        # This file
```

## ⚡ Next Steps

- [ ] Add support for more programming languages in analysis
- [ ] Implement webhook signature validation
- [ ] Add configuration file support beyond environment variables
- [ ] Create proper GitHub App (vs Personal Access Token)
- [ ] Add metrics and monitoring
- [ ] Support for custom review templates per repository
- [ ] Integration with team coding standards and style guides

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with a real PR
5. Submit a pull request (and watch Cyclone review it! 🌪️)

**Built with ❤️ by the ecoplanet engineering team 🌱** 🌪️