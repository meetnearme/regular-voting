package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
	conn *websocket.Conn
	request *http.Request
}

var (
	db       *sql.DB
	mutex    sync.Mutex
	clients  = make(map[*Client]bool)
	upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
	}
	webAuthn *webauthn.WebAuthn
	sessionStore = make(map[string]webauthn.SessionData)
	voterClients = make(map[*Client]bool)
	adminClients = make(map[*Client]bool)
)


type VoteItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	RoundID  int    `json:"roundId"`
	Votes    int    `json:"votes"`
}

type VoteRound struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type VoteMessage struct {
	Type    string    `json:"type"`
	ItemID  int       `json:"itemId,omitempty"`
	Results []VoteItem `json:"results,omitempty"`
}

type User struct {
	ID           string
}

type VoteRecord struct {
	RoundName string `json:"roundName"`
	UserId    string `json:"userId"`
	VoteName  string `json:"voteName"`
}


func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
					w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
			}

			payload, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
			pair := strings.SplitN(string(payload), ":", 2)

			if len(pair) != 2 || !checkCredentials(pair[0], pair[1]) {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
			}

			next.ServeHTTP(w, r)
	}
}

func checkCredentials(username, password string) bool {
	// Replace with secure credential checking, possibly against a database
	return username == "admin" && password == "secret"
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./votes.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create votes table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vote_rounds (
			id INTEGER PRIMARY KEY,
			name TEXT
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vote_items (
			id INTEGER PRIMARY KEY,
			name TEXT,
			round_id INTEGER,
			votes INTEGER DEFAULT 0,
			FOREIGN KEY (round_id) REFERENCES vote_rounds(id)
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create user_votes table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_votes (
			user_id TEXT,
			round_id INTEGER,
			PRIMARY KEY (user_id, round_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (round_id) REFERENCES vote_rounds(id)
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize WebAuthn
	webAuthn, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Voting App",
		RPID:          "localhost",
		RPOrigin:      "http://localhost:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Initialize vote items if not exists
	initializeVoteItems()

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/voter/ws", handleVoterWebSocket)
	http.HandleFunc("/vote", handleVote)
	http.HandleFunc("/admin", basicAuth(handleAdmin))
	http.HandleFunc("/admin/ws", handleAdminWebSocket)
	http.HandleFunc("/admin/vote-records", handleAdminVoteRecords)


	fmt.Printf("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleVoterWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
			log.Printf("Error upgrading to WebSocket: %v", err)
			return
	}
	defer conn.Close()

	client := &Client{conn: conn, request: r}
	voterClients[client] = true
	defer delete(voterClients, client)

	// Send initial vote data
	sendVoteUpdate()

	// Keep connection alive and handle any incoming messages
	for {
			_, _, err := conn.ReadMessage()
			if err != nil {
					break
			}
	}
}

func handleAdminWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	client := &Client{conn: conn, request: r}
	adminClients[client] = true
	defer delete(adminClients, client)

	// Send initial vote records
	sendVoteRecords(conn)

	// Keep connection alive and handle any incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func handleAdminVoteRecords(w http.ResponseWriter, r *http.Request) {
	records, err := getVoteRecords()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(records)
}


func getVoteRecords() ([]VoteRecord, error) {
	rows, err := db.Query(`
			SELECT vr.name AS round_name, uv.user_id, vi.name AS vote_name
			FROM user_votes uv
			JOIN vote_rounds vr ON uv.round_id = vr.id
			JOIN vote_items vi ON uv.round_id = vi.round_id AND vi.id = (
					SELECT vote_items.id
					FROM vote_items
					JOIN user_votes ON user_votes.round_id = vote_items.round_id
					WHERE user_votes.user_id = uv.user_id AND user_votes.round_id = uv.round_id
					LIMIT 1
			)
			ORDER BY vr.id, uv.user_id
	`)
	if err != nil {
			return nil, err
	}
	defer rows.Close()

	var records []VoteRecord
	for rows.Next() {
			var record VoteRecord
			err := rows.Scan(&record.RoundName, &record.UserId, &record.VoteName)
			if err != nil {
					return nil, err
			}
			records = append(records, record)
	}
	return records, nil
}

func sendVoteRecords(conn *websocket.Conn) {
	records, err := getVoteRecords()
	if err != nil {
		log.Printf("Error getting vote records: %v", err)
		return
	}

	err = conn.WriteJSON(records)
	if err != nil {
		log.Printf("Error sending vote records: %v", err)
	}
}

func initializeVoteItems() {
	rounds := []VoteRound{
		{ID: 1, Name: "Round 1"},
		{ID: 2, Name: "Round 2"},
	}

	items := []VoteItem{
		{ID: 1, Name: "Team A", RoundID: 1},
		{ID: 2, Name: "Team B", RoundID: 1},
		{ID: 3, Name: "Team C", RoundID: 2},
		{ID: 4, Name: "Team D", RoundID: 2},
	}

	for _, round := range rounds {
		db.Exec("INSERT OR IGNORE INTO vote_rounds (id, name) VALUES (?, ?)", round.ID, round.Name)
	}

	for _, item := range items {
		db.Exec("INSERT OR IGNORE INTO vote_items (id, name, round_id, votes) VALUES (?, ?, ?, 0)", item.ID, item.Name, item.RoundID)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	component := Home(r)
	templ.Handler(component).ServeHTTP(w, r)
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	component := Admin()
	templ.Handler(component).ServeHTTP(w, r)
}

func handleVote(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var vote struct {
			ItemID       int    `json:"itemId"`
			UserIdEmail  string `json:"userIdEmail"`
		}

		err := json.NewDecoder(r.Body).Decode(&vote)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get the round_id for the voted item
		var roundID int
		err = db.QueryRow("SELECT round_id FROM vote_items WHERE id = ?", vote.ItemID).Scan(&roundID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if user has already voted in this round
		var hasVoted bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM user_votes WHERE user_id = ? AND round_id = ?)", vote.UserIdEmail, roundID).Scan(&hasVoted)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if hasVoted {
			http.Error(w, "User has already voted in this round", http.StatusForbidden)
			return
		}

		// Record the vote
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("UPDATE vote_items SET votes = votes + 1 WHERE id = ?", vote.ItemID)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT OR IGNORE INTO users (id) VALUES (?)", vote.UserIdEmail)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO user_votes (user_id, round_id) VALUES (?, ?)", vote.UserIdEmail, roundID)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tx.Commit()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Broadcast update to all clients
		go sendVoteUpdate()

		w.WriteHeader(http.StatusOK)
	} else if r.Method == "GET" {
		// Handle GET request to fetch current votes
		results, err := getVoteResults()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(VoteMessage{
			Type:    "update",
			Results: results,
		})
	}
}

func sendVoteUpdate() {
	results, err := getVoteResults()
	if err != nil {
			log.Printf("Error getting vote results: %v", err)
			return
	}

	message := VoteMessage{
			Type:    "update",
			Results: results,
	}

	messageJSON, _ := json.Marshal(message)

	mutex.Lock()
	defer mutex.Unlock()
	for client := range voterClients {
			err := client.conn.WriteMessage(websocket.TextMessage, messageJSON)
			if err != nil {
					log.Printf("Error sending to voter client: %v", err)
					client.conn.Close()
					delete(voterClients, client)
			}
	}

	// Send update to admin connections
	records, err := getVoteRecords()
	if err != nil {
			log.Printf("Error getting vote records: %v", err)
			return
	}

	adminMessage, _ := json.Marshal(records)

	for client := range adminClients {
			err := client.conn.WriteMessage(websocket.TextMessage, adminMessage)
			if err != nil {
					log.Printf("Error sending to admin client: %v", err)
					client.conn.Close()
					delete(adminClients, client)
			}
	}
}

func getVoteResults() ([]VoteItem, error) {
	rows, err := db.Query(`
		SELECT vi.id, vi.name, vi.round_id, vi.votes, vr.name
		FROM vote_items vi
		JOIN vote_rounds vr ON vi.round_id = vr.id
		ORDER BY vi.round_id, vi.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []VoteItem
	for rows.Next() {
		var item VoteItem
		var roundName string
		err := rows.Scan(&item.ID, &item.Name, &item.RoundID, &item.Votes, &roundName)
		if err != nil {
			return nil, err
		}
		item.Name = fmt.Sprintf("%s - %s", roundName, item.Name)
		results = append(results, item)
	}
	return results, nil
}
