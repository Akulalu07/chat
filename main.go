package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

var db *sql.DB

const (
	host     = "127.0.0.1"
	port     = 5432
	username = "counter"
	password = "counter"
	dbname   = "counter"
)

func createTable() {
	query := `
    CREATE TABLE IF NOT EXISTS notes(
        id bigserial primary key,
		username text,
		note text
    );`

	_, err := db.Exec(query)
	if err != nil {
		panic(err)
	}

	fmt.Println("Table created successfully")
}
func AddNote(username string, text string) {
	query := `
	INSERT INTO notes(username,note) VALUES ($1,$2)
	`
	_, err := db.Exec(query, username, text)

	if err != nil {
		panic(err)
	}

}

type Note struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Note     string `json:"note"`
}

func TakeFirst(count int) ([]Note, error) {
	query := `
    SELECT id, username, note FROM notes  ORDER BY id DESC LIMIT $1
    `
	rows, err := db.Query(query, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.Id, &note.Username, &note.Note); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

func TakeSomeOld(count int, someid int) ([]Note, error) {
	query := `
    SELECT id, username, note FROM notes WHERE id < $1 ORDER BY id DESC LIMIT $2
    `
	rows, err := db.Query(query, someid, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.Id, &note.Username, &note.Note); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil
}

func TakeAllNew(someid int) ([]Note, error) {
	query := `
    SELECT id, username, note FROM notes WHERE id > $1  ORDER BY id
    `
	rows, err := db.Query(query, someid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.Id, &note.Username, &note.Note); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	//fmt.Println("hello:", notes)
	return notes, nil
}

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

	for id, ch := range c.channels {
		log.Printf("NOTIFY TO %v", id)
		ch <- struct{}{}
	}

	log.Print("CLEARING ClientMap")
	c.channels = make(ClientMap)
}

func (c *Clients) NewClient() chan struct{} {
	// сам напиши
	ch := make(chan struct{}, 1)
	c.Lock()
	defer c.Unlock()
	c.channels[c.counter] = ch
	log.Printf("NEW CLIENT %v", c.counter)
	c.counter++
	return ch
}

var clients = Clients{
	channels: make(ClientMap),
}

func HandleWS(conn *websocket.Conn) error {
	for {
		client := clients.NewClient()
		<-client
		err := conn.WriteMessage(websocket.TextMessage, []byte("Reload!"))
		if err != nil {
			return err
		}
	}
}

func MainWeb(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/ws" {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)

		if err != nil {
			w.WriteHeader(400)
			log.Printf("Couldn't initialize websocket connection: %v", err)
			return
		}

		err = HandleWS(conn)

		if err != nil {
			log.Print("WS connection broken")
		}
		return
	}

	/*
		Обработать POST-запрос.

		Если в теле слово get, отправить в ответ текущее значение счётчика.
		Если слово increase -- увеличить счётчик.
		В противном случае возвратить пустой ответ с кодом состояния 400.
	*/
	if r.Method == "GET" {
		path := filepath.Join("static", r.URL.Path)
		http.ServeFile(w, r, path)

		return
	}

	if r.Method == "POST" {

		body, err := io.ReadAll(r.Body)

		if err != nil {
			log.Panic(err)
		}
		var command map[string]any
		err = json.Unmarshal([]byte(body), &command)

		if err != nil {
			log.Panic(err)
		}

		action := command["action"]

		if action == "get_notes" {
			logx := command["log"]

			var notes []Note

			if logx == "Takefirst" {
				howmuch, ok := command["howmuch"].(float64)
				if !ok {
					log.Println(command)
					w.WriteHeader(400)
					return
				}
				howmuchid := int(howmuch)
				notes, err = TakeFirst(howmuchid)
			} else if logx == "Takesomelower" {
				fmt.Println(command)
				howmuch, ok := command["howmuch"].(float64)
				if !ok {
					log.Println(command)
					w.WriteHeader(400)
					return
				}
				howmuchid := int(howmuch)
				some, ok := command["someid"].(float64)
				if !ok {
					w.WriteHeader(401)
					return
				}
				someid := int(some)
				if !ok {
					w.WriteHeader(400)
					return
				}
				notes, err = TakeSomeOld(howmuchid, someid)
			} else if logx == "Takesomebigger" {
				//fmt.Println("tekesomenew")
				some, ok := command["someid"].(float64)
				fmt.Println(some, ok, command["someid"])
				if !ok {

					w.WriteHeader(401)
					return
				}
				someid := int(some)
				notes, err = TakeAllNew(someid)
			}
			if err != nil {
				log.Panic(err)
			}
			json.NewEncoder(w).Encode(notes)
			return
		}

		if action == "add_note" {
			switch v := command["message"].(type) {
			case string:
				indd := strings.Index(v, ",")
				username := v[:indd]
				note := v[indd+1:]
				AddNote(username, note)
			}
			clients.Notify()
			json.NewEncoder(w).Encode("")
			return

		}

		// код состояния 400
		w.WriteHeader(400)
		return
	}
}

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, username, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Panic(err)
	}
	fmt.Println("Successfully connected to DataBase!")
	/*
			query := `
		    DROP DATABASE counter
		    `

			_, err = db.Exec(query)
			if err != nil {
				fmt.Println(err)
			}
	*/
	createTable()
	fmt.Println("Server Start on :8080")
	http.HandleFunc("/", MainWeb)
	http.ListenAndServe(":8080", nil)

}