package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Location struct {
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Timestamp  string  `json:"timestamp"`
	DeviceID   string  `json:"device_id"`
	DeviceName string  `json:"device_name"`
	// DeviceModel string  `json:"device_model"`
	// DeviceModel string  `json:"device_model"`
	// DeviceModel string  `json:"device_model"`
}

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Location)          // broadcast channel
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	router := mux.NewRouter()

	// Serve static files from the "icons" directory
	router.PathPrefix("/icons/").Handler(http.StripPrefix("/icons/", http.FileServer(http.Dir("icons"))))

	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/location", locationHandler).Methods("POST")
	router.HandleFunc("/ws", wsHandler)

	go handleMessages()

	fmt.Println("Server started at http://localhost:8889")
	log.Fatal(http.ListenAndServe(":8889", router))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("index.html")
	tmpl.Execute(w, nil)
}

func locationHandler(w http.ResponseWriter, r *http.Request) {
	var loc Location
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&loc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Send location data to the broadcast channel
	broadcast <- loc

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	clients[ws] = true

	for {
		var loc Location
		err := ws.ReadJSON(&loc)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
	}
}

func handleMessages() {
	for {
		loc := <-broadcast
		for client := range clients {
			err := client.WriteJSON(loc)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
