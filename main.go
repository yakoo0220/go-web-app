package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func initDB() {
	var err error
	dsn := "root:7ea5837b98d523dc@tcp(127.0.0.1:3306)/dbname"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	input1 := r.FormValue("input1")
	input2 := r.FormValue("input2")
	input3 := r.FormValue("input3")
	selectMenu := r.FormValue("selectMenu")

	var isUsed int
	err := db.QueryRow("SELECT isused FROM submissions WHERE activation_code = ?", input1).Scan(&isUsed)
	if err == sql.ErrNoRows || isUsed == 1 {
		response := map[string]bool{"success": false}
		json.NewEncoder(w).Encode(response)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("UPDATE submissions SET isused = 1, email = ?, password = ?, selectMenu = ? WHERE activation_code = ?", input2, input3, selectMenu, input1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]bool{"success": true}
	json.NewEncoder(w).Encode(response)
}

func main() {
	initDB()
	defer db.Close()

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/submit", submitHandler)

	fmt.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
