# GoLang Web Framework

A powerful CLI-based web framework for Go that processes HTML templates with JSP-like syntax, built on top of the Echo framework. Features both development mode with live reload and production mode with single binary compilation.

## ✨ Features

- **🔥 ASP/JSP-like Template Syntax**: Support for `<% code %>`, `<%= output %>`, and `<%@include file="..." %>` tags
- **🛣️ Dual Routing System**: File-based routing + XML route configuration
- **📦 Single Binary Compilation**: Compile all templates into a standalone executable
- **🔄 Live File Watching**: Automatic reloading during development
- **🌐 Multiple HTTP Methods**: Support for GET, POST, PUT, DELETE, PATCH, and ANY
- **⚡ High Performance**: Built on Echo web framework
- **🔧 CLI Interface**: Easy-to-use command-line tool
- **🚀 Cross-Platform**: Compile for Linux, macOS, Windows, and ARM

## 🚀 Quick Start

### Installation

1. **Clone the repository**
   ```bash
   git clone gosp git url
   cd gosp
   ```

2. **Initialize Go module**
   ```bash
   go mod init gosp
   go mod tidy
   ```

3. **Build the framework**
   ```bash
   go build -o gosp main.go
   ```

### Basic Usage

1. **Create your web directory**
   ```bash
   mkdir -p root_http/includes
   mkdir -p root_http/pages
   ```

2. **Create a simple homepage**
   ```html
   <!-- root_http/index.html -->
   <!DOCTYPE html>
   <html>
   <head><title>My Website</title></head>
   <body>
       <% greeting = "Hello World!" %>
       <h1><%= greeting %></h1>
       <p>Method: <%= request.method %></p>
       <p>Host: <%= request.host %></p>
   </body>
   </html>
   ```

3. **Run development server**
   ```bash
   ./gosp --root ./root_http --port 8080 --watch
   ```

4. **Visit** `http://localhost:8080`

## 📁 Directory Structure

```
my-project/
├── main.go                     # Framework source code
├── go.mod                      # Go dependencies
├── routes.xml                  # Route configuration
├── webframework               # Development binary
├── webframework-compiled      # Production binary (after compilation)
│
└── root_http/                 # Web root directory
    ├── index.html            # Homepage (/)
    ├── includes/             # Shared components
    │   ├── header.html       # <%@include file="includes/header.html" %>
    │   └── footer.html       # Reusable footer
    ├── pages/                # Static pages
    │   ├── about.html        # /pages/about
    │   └── contact.html      # /pages/contact
    ├── api/                  # API endpoints
    │   └── users.html        # /api/users
    └── admin/                # Admin section
        └── dashboard.html    # /admin/dashboard
```

## 🎯 Template Syntax

### Code Blocks
Execute server-side code:
```html
<% variable = "value" %>
<% userName = "John Doe" %>
<% if (request.method == "POST") { %>
    <!-- This runs only for POST requests -->
<% } %>
```

### Output Variables
Display variables and expressions:
```html
<%= variable %>
<%= request.method %>
<%= query.paramName %>
<%= form.fieldName %>
```

### Include Files
Include other template files:
```html
<%@include file="includes/header.html" %>
<%@include file="../shared/footer.html" %>
```

### Built-in Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `request.method` | HTTP method | GET, POST, PUT, DELETE |
| `request.url` | Full request URL | `/page?param=value` |
| `request.host` | Request host | `localhost:8080` |
| `request.remoteaddr` | Client IP | `127.0.0.1:12345` |
| `query.paramName` | Query parameters | `?name=John` → `query.name` |
| `form.fieldName` | Form data | `<input name="email">` → `form.email` |

## 🛣️ Routes Configuration

Create a `routes.xml` file to customize URL routing and HTTP method restrictions:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<routes>
    <!-- Homepage -->
    <route path="/" file="index.html">
        <methods>GET</methods>
    </route>
    
    <!-- Contact form (GET to show, POST to submit) -->
    <route path="/contact" file="pages/contact.html">
        <methods>GET</methods>
        <methods>POST</methods>
    </route>
    
    <!-- API endpoint with full CRUD -->
    <route path="/api/users" file="api/users.html">
        <methods>GET</methods>    <!-- List users -->
        <methods>POST</methods>   <!-- Create user -->
        <methods>PUT</methods>    <!-- Update user -->
        <methods>DELETE</methods> <!-- Delete user -->
    </route>
    
    <!-- Admin area (restricted to GET) -->
    <route path="/admin" file="admin/dashboard.html">
        <methods>GET</methods>
    </route>
    
    <!-- Flexible webhook (accepts all methods) -->
    <route path="/webhook" file="api/webhook.html">
        <methods>ANY</methods>
    </route>
</routes>
```

### Route Elements

- **`<route>`** - Individual route definition
  - **`path`** - URL path (e.g., `/contact`, `/api/users`)
  - **`file`** - HTML file to serve (relative to root_http/)
- **`<methods>`** - Allowed HTTP methods per route

### HTTP Methods

| Method | Purpose | Example Use |
|--------|---------|-------------|
| `GET` | Retrieve data | Display pages, show forms |
| `POST` | Create/Submit | Form submissions, create records |
| `PUT` | Update data | Edit profiles, update settings |
| `DELETE` | Remove data | Delete users, clear data |
| `PATCH` | Partial update | Modify specific fields |
| `ANY` | All methods | Flexible API endpoints |

## 🔧 CLI Commands

### Development Mode
```bash
# Basic server
./webframework --root ./root_http --port 8080

# With live reload
./webframework --root ./root_http --port 8080 --watch

# Custom config
./webframework --root ./web --config custom-routes.xml --port 3000 --watch
```

### Production Compilation
```bash
# Compile templates into standalone binary
./webframework compile --root ./root_http --config routes.xml --output my-app

# Run compiled binary (no external files needed!)
./my-app --port 8080
```

### CLI Options

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--root` | `-r` | Web root directory | `./root_http` |
| `--config` | `-c` | Route configuration file | `routes.xml` |
| `--port` | `-p` | Server port | `8080` |
| `--watch` | `-w` | Enable file watching | `false` |

## 💡 Example Templates

### Simple Homepage
```html
<!-- root_http/index.html -->
<!DOCTYPE html>
<html>
<head>
    <title>My Website</title>
</head>
<body>
    <%@include file="includes/header.html" %>
    
    <% siteName = "My Awesome Site" %>
    <% version = "1.0.0" %>
    
    <h1>Welcome to <%= siteName %></h1>
    <p>Version: <%= version %></p>
    <p>You're using: <%= request.method %> method</p>
    
    <%@include file="includes/footer.html" %>
</body>
</html>
```

### Contact Form
```html
<!-- root_http/pages/contact.html -->
<!DOCTYPE html>
<html>
<body>
    <% if (request.method == "GET") { %>
        <h1>Contact Us</h1>
        <form method="POST" action="/contact">
            <input type="text" name="name" placeholder="Your Name" required>
            <input type="email" name="email" placeholder="Your Email" required>
            <textarea name="message" placeholder="Your Message" required></textarea>
            <button type="submit">Send Message</button>
        </form>
    <% } else { %>
        <h1>Thank You!</h1>
        <p>Thanks <%= form.name %>, we received your message!</p>
        <p>We'll reply to: <%= form.email %></p>
        <p>Your message: <%= form.message %></p>
    <% } %>
</body>
</html>
```

### API Endpoint
```html
<!-- root_http/api/users.html -->
<% if (request.method == "GET") { %>
    {"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}
<% } else if (request.method == "POST") { %>
    {"message": "User created", "name": "<%= form.name %>"}
<% } else if (request.method == "PUT") { %>
    {"message": "User updated", "name": "<%= form.name %>"}
<% } else if (request.method == "DELETE") { %>
    {"message": "User deleted"}
<% } else { %>
    {"error": "Method not allowed"}
<% } %>
```

## ⚡ Two Deployment Modes

### 🔧 Development Mode
- Templates read from disk at runtime
- Live file watching and hot reload
- Perfect for development and debugging

```bash
./webframework --root ./root_http --port 8080 --watch
```

### 🚀 Production Mode
- All templates compiled into single binary
- No external file dependencies
- Lightning-fast startup and serving

```bash
# Compile
./webframework compile --root ./root_http --output my-app

# Deploy single binary anywhere
./my-app --port 8080
```

## 🔄 URL Routing Examples

### File-based Routing (Automatic)
```
URL: /pages/about        → File: root_http/pages/about.html
URL: /blog/post1         → File: root_http/blog/post1.html
URL: /admin/users        → File: root_http/admin/users.html
```

### Custom Routing (via routes.xml)
```xml
<!-- SEO-friendly URLs -->
<route path="/about" file="pages/about-us.html">
    <methods>GET</methods>
</route>

<!-- API endpoints -->
<route path="/api/users" file="api/users-handler.html">
    <methods>GET</methods>
    <methods>POST</methods>
</route>
```

## 🛠️ Build Commands

### Using Go Commands
```bash
# Development build
go build -o gosp main.go

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o gosp-linux main.go
GOOS=windows GOARCH=amd64 go build -o gosp-windows.exe main.go
GOOS=darwin GOARCH=amd64 go build -o gosp-macos main.go
```

### Dependencies
```bash
# Install dependencies
go mod tidy

# Update dependencies
go get -u github.com/labstack/echo/v4
go get -u github.com/spf13/cobra
go get -u github.com/fsnotify/fsnotify
```

## 🔒 Security Features

### Method Restrictions
```xml
<!-- Only allow safe methods -->
<route path="/admin/data" file="admin/data.html">
    <methods>GET</methods>  <!-- Read-only -->
</route>

<!-- Restrict dangerous operations -->
<route path="/api/delete" file="api/delete.html">
    <methods>DELETE</methods>  <!-- Only DELETE allowed -->
</route>
```

### Built-in Middleware
- **CORS support** - Cross-origin resource sharing
- **Request logging** - All requests logged
- **Panic recovery** - Automatic recovery from errors

## 📋 Requirements

- **Go 1.19+**
- **Dependencies** (auto-installed via `go mod tidy`):
  - `github.com/labstack/echo/v4` - Web framework
  - `github.com/spf13/cobra` - CLI interface
  - `github.com/fsnotify/fsnotify` - File watching

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes
4. Add tests if applicable
5. Commit your changes: `git commit -am 'Add feature'`
6. Push to the branch: `git push origin feature-name`
7. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.



## 🚀 Getting Started Checklist

- [ ] Clone the repository
- [ ] Run `go mod tidy`
- [ ] Build with `go build -o gosp main.go`
- [ ] Create `root_http/` directory
- [ ] Add your HTML templates with JSP-like syntax
- [ ] Create `routes.xml` for custom routing (optional)
- [ ] Run development server: `./gosp --root ./root_http --watch`
- [ ] Compile for production: `./gosp compile --output my-app`
- [ ] Deploy single binary: `./my-app --port 8080`

---

