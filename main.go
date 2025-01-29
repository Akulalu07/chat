package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"server/mycrypto"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

var db *sql.DB

const (
	dbhost     = "127.0.0.1"
	dbport     = 5432
	dbusername = "counter"
	dbpassword = "counter"
	dbname     = "counter"
)

var ErrUserNotFound = errors.New("user not found")

// Database operations

func initDB() error {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbhost, dbport, dbusername, dbpassword, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Successfully connected to DataBase!")
	return nil
}

func initialize() {

}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS notes(
            id bigserial primary key,
            username text,
            note text
        );`,
		`CREATE TABLE IF NOT EXISTS users(
            id bigserial primary key,
            username text UNIQUE,
            salt text,
            sha text
        );`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	fmt.Println("Tables created successfully")
	return nil
}

func dropTables() error {
	queries := []string{
		`DROP TABLE IF EXISTS notes;`,
		`DROP TABLE IF EXISTS users;`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to drop table: %w", err)
		}
	}

	fmt.Println("Tables dropped")
	return nil
}

// User operations

func AddUser(username, salt, password string) error {
	query := `INSERT INTO users(username, salt, sha) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, username, salt, mycrypto.PasswordToHash(password, salt))
	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}
	return nil
}

func GetUser(username string) (string, string, error) {
	query := "SELECT salt, sha FROM users WHERE username = $1;"
	var salt, sha string
	err := db.QueryRow(query, username).Scan(&salt, &sha)
	if err == sql.ErrNoRows {
		return "", "", ErrUserNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}
	return salt, sha, nil
}

// Note operations

func AddNote(username, text string) error {
	query := `INSERT INTO notes(username, note) VALUES ($1, $2)`
	_, err := db.Exec(query, username, text)
	if err != nil {
		return fmt.Errorf("failed to add note: %w", err)
	}
	return nil
}

type Note struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Note     string `json:"note"`
}

func TakeFirst(count int) ([]Note, error) {
	query := `SELECT id, username, note FROM notes ORDER BY id DESC LIMIT $1`
	return fetchNotes(query, count)
}

func TakeSomeOld(count, someid int) ([]Note, error) {
	query := `SELECT id, username, note FROM notes WHERE id < $1 ORDER BY id DESC LIMIT $2`
	return fetchNotes(query, someid, count)
}

func TakeAllNew(someid int) ([]Note, error) {
	query := `SELECT id, username, note FROM notes WHERE id > $1 ORDER BY id`
	return fetchNotes(query, someid)
}

func fetchNotes(query string, args ...interface{}) ([]Note, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.Id, &note.Username, &note.Note); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over notes: %w", err)
	}

	return notes, nil
}

// Client management

type ClientId uint64
type ClientMap map[ClientId]chan struct{}

type Clients struct {
	channels ClientMap
	counter  ClientId
	sync.Mutex
}

func (c *Clients) Notify() {
	c.Lock()
	defer c.Unlock()

	for _, ch := range c.channels {
		ch <- struct{}{}
	}

	c.channels = make(ClientMap)
}

func (c *Clients) NewClient() chan struct{} {
	ch := make(chan struct{}, 1)
	c.Lock()
	defer c.Unlock()
	c.channels[c.counter] = ch
	c.counter++
	return ch
}

var clients = Clients{
	channels: make(ClientMap),
}

// WebSocket handling
func HandleWS(conn *websocket.Conn) error {
	for {
		client := clients.NewClient()
		<-client
		err := conn.WriteMessage(websocket.TextMessage, []byte("Reload!"))
		if err != nil {
			return fmt.Errorf("failed to write WebSocket message: %w", err)
		}
	}
}

// HTTP request handling
func MainWeb(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/ws" {
		handleWebSocket(w, r)
		return
	}

	if r.Method == "GET" {
		handleGetRequest(w, r)
		return
	}

	if r.Method == "POST" {
		handlePostRequest(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Couldn't initialize websocket connection: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = HandleWS(conn)
	if err != nil {
		log.Print("WS connection broken")
	}
}

func handleGetRequest(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join("static", r.URL.Path)
	http.ServeFile(w, r, path)
}

func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var command map[string]interface{}
	err = json.Unmarshal(body, &command)
	if err != nil {
		log.Printf("Failed to unmarshal JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	action, ok := command["action"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch action {
	case "get_notes":
		handleGetNotes(w, command)
	case "add_note":
		handleAddNote(w, command)
	case "registration":
		handleRegistration(w, command)
	case "login":
		handleLogin(w, command)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func handleGetNotes(w http.ResponseWriter, command map[string]interface{}) {
	logx, ok := command["log"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var notes []Note
	var err error

	switch logx {
	case "Takefirst":
		howmuch, ok := command["howmuch"].(float64)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		notes, err = TakeFirst(int(howmuch))
	case "Takesomelower":
		howmuch, ok1 := command["howmuch"].(float64)
		some, ok2 := command["someid"].(float64)
		if !ok1 || !ok2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		notes, err = TakeSomeOld(int(howmuch), int(some))
	case "Takesomebigger":
		some, ok := command["someid"].(float64)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		notes, err = TakeAllNew(int(some))
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Failed to get notes: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(notes)
}

func handleAddNote(w http.ResponseWriter, command map[string]interface{}) {
	message, ok := command["message"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(message, ",", 2)
	if len(parts) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username, note := parts[0], parts[1]
	err := AddNote(username, note)
	if err != nil {
		log.Printf("Failed to add note: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clients.Notify()
	json.NewEncoder(w).Encode("")
}

func handleRegistration(w http.ResponseWriter, command map[string]interface{}) {
	user, ok1 := command["user"].(string)
	pass, ok2 := command["pass"].(string)
	if !ok1 || !ok2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	salt, _, err := GetUser(user)
	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err != ErrUserNotFound {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	salt = mycrypto.Generate_salt()
	err = AddUser(user, salt, pass)
	if err != nil {
		log.Printf("Failed to add user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleLogin(w http.ResponseWriter, command map[string]interface{}) {
	user, ok1 := command["user"].(string)
	pass, ok2 := command["pass"].(string)
	if !ok1 || !ok2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	salt, sha, err := GetUser(user)
	if err == ErrUserNotFound {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if mycrypto.PasswordToHash(pass, salt) != sha {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Utility functions

func GetLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", fmt.Errorf("failed to get addresses: %w", err)
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid local IP found")
}
func main() {
	initialize()
	mycrypto.Init()

	err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	var flag string
	fmt.Println("Enter yes/not for drop table")
	fmt.Scan(&flag)
	if flag == "yes" {
		err = dropTables()
		if err != nil {
			log.Fatalf("Failed to drop tables: %v", err)
		}
	}

	err = createTables()
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	fmt.Println("Server Start on :8080")
	http.HandleFunc("/", MainWeb)

	localIP, err := GetLocalIP()
	if err != nil {
		log.Printf("Failed to get local IP: %v", err)
	} else {
		fmt.Printf("Local IP: %s\n", localIP)
	}

	log.Fatal(http.ListenAndServe(":8080", nil))
}
