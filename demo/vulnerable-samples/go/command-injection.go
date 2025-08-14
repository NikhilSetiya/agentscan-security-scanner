package main

// Vulnerable Go code samples for AgentScan demo
// These examples contain intentional security vulnerabilities for demonstration purposes

import (
	"crypto/md5"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Hardcoded Secrets - VULNERABLE
const (
	APIKey        = "sk-1234567890abcdef"  // VULNERABLE: Hardcoded API key
	DatabaseURL   = "admin:password@tcp(localhost:3306)/myapp" // VULNERABLE: Hardcoded credentials
	JWTSecret     = "my-jwt-secret"        // VULNERABLE: Hardcoded JWT secret
)

// SQL Injection Vulnerabilities
func getUserByID(db *sql.DB, userID string) (*User, error) {
	// VULNERABLE: Direct string concatenation in SQL query
	query := "SELECT id, username, email FROM users WHERE id = " + userID
	
	row := db.QueryRow(query)
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func authenticateUser(db *sql.DB, username, password string) bool {
	// VULNERABLE: String formatting in SQL query
	query := fmt.Sprintf("SELECT id FROM users WHERE username = '%s' AND password = '%s'", username, password)
	
	var id int
	err := db.QueryRow(query).Scan(&id)
	return err == nil
}

func searchUsers(db *sql.DB, searchTerm string) ([]User, error) {
	// VULNERABLE: Direct interpolation in SQL query
	query := "SELECT id, username, email FROM users WHERE username LIKE '%" + searchTerm + "%'"
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var user User
		rows.Scan(&user.ID, &user.Username, &user.Email)
		users = append(users, user)
	}
	return users, nil
}

// Command Injection Vulnerabilities
func pingHandler(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		host = "localhost"
	}
	
	// VULNERABLE: Direct command execution with user input
	cmd := exec.Command("sh", "-c", "ping -c 1 "+host)
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Ping failed", http.StatusInternalServerError)
		return
	}
	
	fmt.Fprintf(w, "<pre>%s</pre>", string(output))
}

func backupHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	
	// VULNERABLE: Command injection through exec.Command
	cmd := exec.Command("cp", filename, "/backup/")
	err := cmd.Run()
	if err != nil {
		http.Error(w, "Backup failed", http.StatusInternalServerError)
		return
	}
	
	fmt.Fprintf(w, "Backup completed for %s", filename)
}

func executeSystemCommand(command string) (string, error) {
	// VULNERABLE: Using shell execution with user input
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.Output()
	return string(output), err
}

// Path Traversal Vulnerabilities
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	
	// VULNERABLE: Direct file access without path validation
	filePath := filepath.Join("/uploads", filename)
	
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(content)
}

func readConfigFile(configName string) ([]byte, error) {
	// VULNERABLE: Path traversal through string concatenation
	configPath := "/etc/myapp/" + configName
	return ioutil.ReadFile(configPath)
}

func serveStaticFile(w http.ResponseWriter, r *http.Request) {
	// VULNERABLE: No path validation
	requestedFile := r.URL.Path[1:] // Remove leading slash
	http.ServeFile(w, r, requestedFile)
}

// Weak Cryptography
func hashPassword(password string) string {
	// VULNERABLE: Using MD5 for password hashing
	hash := md5.Sum([]byte(password))
	return fmt.Sprintf("%x", hash)
}

func generateToken() string {
	// VULNERABLE: Using weak random number generation
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Intn(999999))
}

func weakEncryption(data []byte, key string) []byte {
	// VULNERABLE: Simple XOR encryption (not secure)
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ key[i%len(key)]
	}
	return result
}

// Template Injection Vulnerabilities
func profileHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("name")
	if username == "" {
		username = "Guest"
	}
	
	// VULNERABLE: Direct template execution with user input
	tmplStr := fmt.Sprintf("<h1>Welcome %s!</h1>", username)
	tmpl, err := template.New("profile").Parse(tmplStr)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	
	tmpl.Execute(w, nil)
}

func renderTemplate(w http.ResponseWriter, r *http.Request) {
	templateContent := r.URL.Query().Get("template")
	
	// VULNERABLE: Parsing and executing user-controlled template
	tmpl, err := template.New("user").Parse(templateContent)
	if err != nil {
		http.Error(w, "Invalid template", http.StatusBadRequest)
		return
	}
	
	tmpl.Execute(w, map[string]string{"user": "admin"})
}

// XML External Entity (XXE) Vulnerability
type XMLData struct {
	XMLName xml.Name `xml:"data"`
	Content string   `xml:",chardata"`
}

func parseXMLHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	
	var data XMLData
	// VULNERABLE: XML parsing without disabling external entities
	err = xml.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Invalid XML", http.StatusBadRequest)
		return
	}
	
	fmt.Fprintf(w, "Parsed content: %s", data.Content)
}

// Race Condition Vulnerability
var (
	balance = 1000
	// VULNERABLE: No proper synchronization
)

func withdrawMoney(amount int) bool {
	// VULNERABLE: Race condition - check and update without locking
	if balance >= amount {
		time.Sleep(100 * time.Millisecond) // Simulate processing
		balance -= amount
		return true
	}
	return false
}

func transferHandler(w http.ResponseWriter, r *http.Request) {
	amountStr := r.URL.Query().Get("amount")
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	
	if withdrawMoney(amount) {
		fmt.Fprintf(w, "Withdrawal successful. New balance: %d", balance)
	} else {
		fmt.Fprintf(w, "Insufficient funds")
	}
}

// Information Disclosure
func debugHandler(w http.ResponseWriter, r *http.Request) {
	// VULNERABLE: Exposing sensitive debug information
	fmt.Fprintf(w, "Environment Variables:\n")
	for _, env := range os.Environ() {
		fmt.Fprintf(w, "%s\n", env)
	}
	
	fmt.Fprintf(w, "\nCurrent Directory: %s\n", getCurrentDir())
	fmt.Fprintf(w, "Process ID: %d\n", os.Getpid())
}

func getCurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}

// Insecure File Upload
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	// VULNERABLE: No file type validation or size limits
	filename := header.Filename
	
	content, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	
	// VULNERABLE: Writing file without validation
	err = ioutil.WriteFile(filepath.Join("/uploads", filename), content, 0644)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	
	fmt.Fprintf(w, "File %s uploaded successfully", filename)
}

// Insecure Direct Object Reference
func documentHandler(w http.ResponseWriter, r *http.Request) {
	docID := strings.TrimPrefix(r.URL.Path, "/document/")
	
	// VULNERABLE: No authorization check
	filePath := filepath.Join("/documents", docID+".txt")
	
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	
	w.Write(content)
}

// Unsafe Reflection
func callFunction(functionName string, args []interface{}) interface{} {
	// VULNERABLE: Dynamic function calling without validation
	// This is a simplified example - real reflection vulnerabilities are more complex
	switch functionName {
	case "getUserByID":
		if len(args) >= 2 {
			db := args[0].(*sql.DB)
			userID := args[1].(string)
			user, _ := getUserByID(db, userID)
			return user
		}
	case "executeSystemCommand":
		if len(args) >= 1 {
			command := args[0].(string)
			result, _ := executeSystemCommand(command)
			return result
		}
	}
	return nil
}

// User struct for examples
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Main function with vulnerable server setup
func main() {
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/backup", backupHandler)
	http.HandleFunc("/download", downloadHandler)
	http.HandleFunc("/profile", profileHandler)
	http.HandleFunc("/render", renderTemplate)
	http.HandleFunc("/xml", parseXMLHandler)
	http.HandleFunc("/transfer", transferHandler)
	http.HandleFunc("/debug", debugHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/document/", documentHandler)
	http.HandleFunc("/", serveStaticFile)
	
	// VULNERABLE: Running server without proper security headers or HTTPS
	log.Println("Starting vulnerable server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}