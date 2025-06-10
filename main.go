package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/fsnotify/fsnotify"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
)

// Route configuration structure
type RouteConfig struct {
	XMLName xml.Name `xml:"routes"`
	Routes  []Route  `xml:"route"`
}

type Route struct {
	Path    string   `xml:"path,attr"`
	File    string   `xml:"file,attr"`
	Methods []string `xml:"methods"`
}

// Template processor for JSP-like syntax
type TemplateProcessor struct {
	rootPath string
	data     map[string]interface{}
	embedded bool
}

// File watcher
type FileWatcher struct {
	watcher  *fsnotify.Watcher
	rootPath string
	server   *echo.Echo
}

var (
	rootPath   string
	configFile string
	port       string
	watch      bool
	output     string
	embedded   bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "webframework",
		Short: "GoLang Web Framework with JSP-like template processing",
		Long:  "A CLI tool that serves HTML templates with JSP-like syntax using Echo framework",
		Run:   runServer,
	}

	var compileCmd = &cobra.Command{
		Use:   "compile",
		Short: "Compile HTML templates into a Go binary",
		Long:  "Generate a standalone Go binary with embedded templates and routes",
		Run:   compileTemplates,
	}

	// Server flags
	rootCmd.Flags().StringVarP(&rootPath, "root", "r", "./root_http", "Root directory for web files")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "routes.xml", "XML configuration file for routing")
	rootCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to run the server on")
	rootCmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for file changes and reload")
	rootCmd.Flags().BoolVarP(&embedded, "embedded", "e", false, "Run with embedded templates (compiled mode)")

	// Compile flags
	compileCmd.Flags().StringVarP(&rootPath, "root", "r", "./root_http", "Root directory for web files")
	compileCmd.Flags().StringVarP(&configFile, "config", "c", "routes.xml", "XML configuration file for routing")
	compileCmd.Flags().StringVarP(&output, "output", "o", "webframework-compiled", "Output binary name")

	rootCmd.AddCommand(compileCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Load routes configuration
	routes, err := loadRouteConfig(configFile)
	if err != nil {
		log.Printf("Warning: Could not load route config: %v", err)
		routes = &RouteConfig{}
	}

	// Setup routes
	setupRoutes(e, routes)

	// Setup file watcher if enabled
	if watch {
		watcher, err := setupFileWatcher(rootPath, e, routes)
		if err != nil {
			log.Printf("Warning: Could not setup file watcher: %v", err)
		} else {
			defer watcher.watcher.Close()
			go watcher.watchFiles()
		}
	}

	// Start server
	log.Printf("Server starting on port %s", port)
	log.Printf("Root directory: %s", rootPath)
	log.Printf("Config file: %s", configFile)
	log.Printf("File watching: %v", watch)

	e.Logger.Fatal(e.Start(":" + port))
}

func loadRouteConfig(configPath string) (*RouteConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config RouteConfig
	err = xml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func setupRoutes(e *echo.Echo, routes *RouteConfig) {
	// Setup configured routes
	for _, route := range routes.Routes {
		for _, method := range route.Methods {
			switch strings.ToUpper(method) {
			case "GET":
				e.GET(route.Path, createHandler(route.File))
			case "POST":
				e.POST(route.Path, createHandler(route.File))
			case "PUT":
				e.PUT(route.Path, createHandler(route.File))
			case "DELETE":
				e.DELETE(route.Path, createHandler(route.File))
			case "PATCH":
				e.PATCH(route.Path, createHandler(route.File))
			case "ANY":
				e.Any(route.Path, createHandler(route.File))
			}
		}
	}

	// Setup catch-all route for file-based routing
	e.Any("/*", fileBasedHandler)
}

func createHandler(filename string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return processTemplate(c, filename)
	}
}

func fileBasedHandler(c echo.Context) error {
	path := c.Request().URL.Path
	if path == "/" {
		path = "/index"
	}

	// Remove leading slash and add .html extension
	filename := strings.TrimPrefix(path, "/") + ".html"

	return processTemplate(c, filename)
}

func processTemplate(c echo.Context, filename string) error {
	fullPath := filepath.Join(rootPath, filename)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return c.String(http.StatusNotFound, "File not found: "+filename)
	}

	// Read template file
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error reading file: "+err.Error())
	}

	// Process JSP-like tags
	processor := &TemplateProcessor{
		rootPath: rootPath,
		data:     make(map[string]interface{}),
		embedded: false,
	}

	// Add request data to template context
	processor.data["request"] = c.Request()
	processor.data["params"] = c.ParamValues()
	processor.data["query"] = c.QueryParams()
	processor.data["form"] = c.Request().Form

	processedContent, err := processor.processTemplate(string(content), c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Template processing error: "+err.Error())
	}

	return c.HTML(http.StatusOK, processedContent)
}

func (tp *TemplateProcessor) processTemplate(content string, c echo.Context) (string, error) {
	// Process include tags first
	content = tp.processIncludes(content)

	// Process code expression tags <%...%>
	content = tp.processCodeExpressions(content, c)

	// Process output tags <%=...%>
	content = tp.processOutputTags(content, c)

	return content, nil
}

func (tp *TemplateProcessor) processIncludes(content string) string {
	includeRegex := regexp.MustCompile(`<%@include\s+file="([^"]+)"\s*%>`)

	return includeRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := includeRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		includeFile := matches[1]
		includePath := filepath.Join(tp.rootPath, includeFile)

		includeContent, err := ioutil.ReadFile(includePath)
		if err != nil {
			return fmt.Sprintf("<!-- Include error: %v -->", err)
		}

		// Recursively process includes
		return tp.processIncludes(string(includeContent))
	})
}

func (tp *TemplateProcessor) processCodeExpressions(content string, c echo.Context) string {
	codeRegex := regexp.MustCompile(`<%\s*([^=][^%]*)\s*%>`)

	return codeRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := codeRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		code := strings.TrimSpace(matches[1])

		// Simple variable assignment processing
		if strings.Contains(code, "=") {
			parts := strings.SplitN(code, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				varValue := strings.TrimSpace(parts[1])

				// Remove quotes if present
				if strings.HasPrefix(varValue, "\"") && strings.HasSuffix(varValue, "\"") {
					varValue = varValue[1 : len(varValue)-1]
				}

				tp.data[varName] = varValue
			}
		}

		return "" // Code blocks don't output content
	})
}

func (tp *TemplateProcessor) processOutputTags(content string, c echo.Context) string {
	outputRegex := regexp.MustCompile(`<%=\s*([^%]+)\s*%>`)

	return outputRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := outputRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		expression := strings.TrimSpace(matches[1])

		// Handle simple variable output
		if value, exists := tp.data[expression]; exists {
			return fmt.Sprintf("%v", value)
		}

		// Handle request parameters
		if strings.HasPrefix(expression, "request.") {
			return tp.handleRequestExpression(expression, c)
		}

		// Handle query parameters
		if strings.HasPrefix(expression, "query.") {
			paramName := strings.TrimPrefix(expression, "query.")
			return c.QueryParam(paramName)
		}

		// Handle form parameters
		if strings.HasPrefix(expression, "form.") {
			paramName := strings.TrimPrefix(expression, "form.")
			return c.FormValue(paramName)
		}

		// Handle simple expressions (this uses strconv)
		if strings.Contains(expression, "+") {
			return tp.evaluateSimpleExpression(expression)
		}

		return expression // Return as-is if not recognized
	})
}

func (tp *TemplateProcessor) handleRequestExpression(expression string, c echo.Context) string {
	switch expression {
	case "request.method":
		return c.Request().Method
	case "request.url":
		return c.Request().URL.String()
	case "request.host":
		return c.Request().Host
	case "request.remoteaddr":
		return c.Request().RemoteAddr
	default:
		return expression
	}
}

func (tp *TemplateProcessor) evaluateSimpleExpression(expression string) string {
	// Simple arithmetic evaluation (this function uses strconv)
	parts := strings.Split(expression, "+")
	if len(parts) == 2 {
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		// Try numeric addition
		if leftVal, err1 := strconv.Atoi(left); err1 == nil {
			if rightVal, err2 := strconv.Atoi(right); err2 == nil {
				return strconv.Itoa(leftVal + rightVal)
			}
		}

		// String concatenation fallback
		return left + right
	}

	// Try simple number parsing for single values
	if val, err := strconv.Atoi(strings.TrimSpace(expression)); err == nil {
		return strconv.Itoa(val)
	}

	return expression
}

func setupFileWatcher(rootPath string, server *echo.Echo, routes *RouteConfig) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw := &FileWatcher{
		watcher:  watcher,
		rootPath: rootPath,
		server:   server,
	}

	// Add root directory to watcher
	err = watcher.Add(rootPath)
	if err != nil {
		return nil, err
	}

	// Add all subdirectories
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return fw, nil
}

func (fw *FileWatcher) watchFiles() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("File modified: %s", event.Name)
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("File created: %s", event.Name)
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					fw.watcher.Add(event.Name)
				}
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// COMPILATION FUNCTIONS

func compileTemplates(cmd *cobra.Command, args []string) {
	log.Printf("🔥 Compiling templates from: %s", rootPath)
	log.Printf("📄 Config file: %s", configFile)
	log.Printf("📦 Output binary: %s", output)

	// Scan all HTML files
	templates := make(map[string]string)
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			// Get relative path from root
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}

			// Convert to forward slashes for consistency
			relPath = filepath.ToSlash(relPath)

			// Read file content
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			templates[relPath] = string(content)
			log.Printf("✅ Added template: %s", relPath)
		}

		return nil
	})

	if err != nil {
		log.Fatal("❌ Error scanning templates:", err)
	}

	// Load routes configuration
	routes, err := loadRouteConfig(configFile)
	if err != nil {
		log.Printf("⚠️  Warning: Could not load route config: %v", err)
		routes = &RouteConfig{}
	}

	// Generate compiled binary
	err = generateCompiledBinary(templates, routes, output)
	if err != nil {
		log.Fatal("❌ Error generating binary:", err)
	}

	log.Printf("🎉 Successfully compiled %d templates into %s", len(templates), output)
	log.Printf("🚀 Run with: ./%s --port 8080", output)
}

func generateCompiledBinary(templates map[string]string, routes *RouteConfig, outputPath string) error {
	// Create temporary directory
	tempDir, err := ioutil.TempDir("", "webframework-compile-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	log.Printf("🔧 Using temporary directory: %s", tempDir)

	// Generate main.go
	mainGoPath := filepath.Join(tempDir, "main.go")
	err = generateMainGo(templates, routes, mainGoPath)
	if err != nil {
		return fmt.Errorf("failed to generate main.go: %v", err)
	}

	// Generate go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	err = generateGoMod(goModPath)
	if err != nil {
		return fmt.Errorf("failed to generate go.mod: %v", err)
	}

	// Build the binary
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute output path: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	err = os.Chdir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Download dependencies
	log.Println("📦 Downloading dependencies...")
	err = executeCommand("go mod tidy")
	if err != nil {
		return fmt.Errorf("failed to download dependencies: %v", err)
	}

	// Build the binary
	log.Printf("🔨 Building binary: %s", absOutputPath)
	buildCmd := fmt.Sprintf("go build -o %s main.go", absOutputPath)
	err = executeCommand(buildCmd)
	if err != nil {
		return fmt.Errorf("failed to build binary: %v", err)
	}

	return nil
}

func generateGoMod(outputPath string) error {
	goModContent := `module compiled-webframework

go 1.19

require (
	github.com/labstack/echo/v4 v4.11.1
	github.com/spf13/cobra v1.7.0
)

require (
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/labstack/gommon v0.4.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/time v0.3.0 // indirect
)
`

	return ioutil.WriteFile(outputPath, []byte(goModContent), 0644)
}

func generateMainGo(templates map[string]string, routes *RouteConfig, outputPath string) error {
	// Create the template data structure
	data := struct {
		Templates map[string]string
		Routes    *RouteConfig
	}{
		Templates: templates,
		Routes:    routes,
	}

	// Create the template
	tmpl := template.New("main")
	tmpl, err := tmpl.Parse(compiledMainTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// Execute template
	err = tmpl.Execute(file, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

func executeCommand(cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	command := exec.Command(parts[0], parts[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}

// Template for the compiled binary - SIMPLIFIED VERSION
const compiledMainTemplate = `package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
)

type RouteConfig struct {
	XMLName xml.Name ` + "`xml:\"routes\"`" + `
	Routes  []Route  ` + "`xml:\"route\"`" + `
}

type Route struct {
	Path    string   ` + "`xml:\"path,attr\"`" + `
	File    string   ` + "`xml:\"file,attr\"`" + `
	Methods []string ` + "`xml:\"methods\"`" + `
}

type TemplateProcessor struct {
	data map[string]interface{}
}

var embeddedTemplates = map[string]string{
{{range $key, $value := .Templates}}	{{printf "%q" $key}}: {{printf "%q" $value}},
{{end}}}

var embeddedRoutes = &RouteConfig{
	Routes: []Route{
{{range .Routes.Routes}}		{
			Path: {{printf "%q" .Path}},
			File: {{printf "%q" .File}},
			Methods: []string{ {{range .Methods}}{{printf "%q" .}}, {{end}} },
		},
{{end}}	},
}

var port string

func main() {
	var rootCmd = &cobra.Command{
		Use:   "compiled-webframework",
		Short: "Compiled GoLang Web Framework",
		Run:   runServer,
	}
	rootCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to run the server on")
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	setupRoutes(e, embeddedRoutes)
	log.Printf("🚀 Compiled server starting on port %s with %d templates", port, len(embeddedTemplates))
	e.Logger.Fatal(e.Start(":" + port))
}

func setupRoutes(e *echo.Echo, routes *RouteConfig) {
	for _, route := range routes.Routes {
		for _, method := range route.Methods {
			switch strings.ToUpper(method) {
			case "GET":
				e.GET(route.Path, createHandler(route.File))
			case "POST":
				e.POST(route.Path, createHandler(route.File))
			case "PUT":
				e.PUT(route.Path, createHandler(route.File))
			case "DELETE":
				e.DELETE(route.Path, createHandler(route.File))
			case "ANY":
				e.Any(route.Path, createHandler(route.File))
			}
		}
	}
	e.Any("/*", fileBasedHandler)
}

func createHandler(filename string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return processTemplate(c, filename)
	}
}

func fileBasedHandler(c echo.Context) error {
	path := c.Request().URL.Path
	if path == "/" {
		path = "/index"
	}
	filename := strings.TrimPrefix(path, "/") + ".html"
	return processTemplate(c, filename)
}

func processTemplate(c echo.Context, filename string) error {
	content, exists := embeddedTemplates[filename]
	if !exists {
		return c.String(http.StatusNotFound, "Template not found: "+filename)
	}
	processor := &TemplateProcessor{data: make(map[string]interface{})}
	processor.data["request"] = c.Request()
	processor.data["query"] = c.QueryParams()
	processor.data["form"] = c.Request().Form
	processedContent, err := processor.processTemplate(content, c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Template processing error: "+err.Error())
	}
	return c.HTML(http.StatusOK, processedContent)
}

func (tp *TemplateProcessor) processTemplate(content string, c echo.Context) (string, error) {
	content = tp.processIncludes(content)
	content = tp.processCodeExpressions(content, c)
	content = tp.processOutputTags(content, c)
	return content, nil
}

func (tp *TemplateProcessor) processIncludes(content string) string {
	includeRegex := regexp.MustCompile(` + "`<%@include\\s+file=\"([^\"]+)\"\\s*%>`)" + `
	return includeRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := includeRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		includeFile := matches[1]
		includeContent, exists := embeddedTemplates[includeFile]
		if !exists {
			return "<!-- Include error: template " + includeFile + " not found -->"
		}
		return tp.processIncludes(includeContent)
	})
}

func (tp *TemplateProcessor) processCodeExpressions(content string, c echo.Context) string {
	codeRegex := regexp.MustCompile(` + "`<%\\s*([^=][^%]*)\\s*%>`)" + `
	return codeRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := codeRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		code := strings.TrimSpace(matches[1])
		if strings.Contains(code, "=") {
			parts := strings.SplitN(code, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				varValue := strings.TrimSpace(parts[1])
				if strings.HasPrefix(varValue, "\"") && strings.HasSuffix(varValue, "\"") {
					varValue = varValue[1 : len(varValue)-1]
				}
				tp.data[varName] = varValue
			}
		}
		return ""
	})
}

func (tp *TemplateProcessor) processOutputTags(content string, c echo.Context) string {
	outputRegex := regexp.MustCompile(` + "`<%=\\s*([^%]+)\\s*%>`)" + `
	return outputRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := outputRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		expression := strings.TrimSpace(matches[1])
		if value, exists := tp.data[expression]; exists {
			return fmt.Sprintf("%v", value)
		}
		if strings.HasPrefix(expression, "request.") {
			return tp.handleRequestExpression(expression, c)
		}
		if strings.HasPrefix(expression, "query.") {
			paramName := strings.TrimPrefix(expression, "query.")
			return c.QueryParam(paramName)
		}
		if strings.HasPrefix(expression, "form.") {
			paramName := strings.TrimPrefix(expression, "form.")
			return c.FormValue(paramName)
		}
		return expression
	})
}

func (tp *TemplateProcessor) handleRequestExpression(expression string, c echo.Context) string {
	switch expression {
	case "request.method":
		return c.Request().Method
	case "request.url":
		return c.Request().URL.String()
	case "request.host":
		return c.Request().Host
	default:
		return expression
	}
}
`
