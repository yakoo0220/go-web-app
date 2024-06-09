package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var (
	templates = template.Must(template.ParseFiles("templates/login.html", "templates/dashboard.html"))
	db        *sql.DB
)

func main() {
	// 设置日志文件
	logFile, err := os.OpenFile("/www/wwwroot/go-web-app/go.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	// 数据库连接
	db, err = sql.Open("mysql", "root:5f3831a5b18f4955@tcp(127.0.0.1:3306)/go_web_app")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", loginHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		activationCode := r.FormValue("activationCode")

		if validateAndUseActivationCode(activationCode) {
			http.SetCookie(w, &http.Cookie{
				Name:  "session_token",
				Value: "some_session_token",
			})
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		} else {
			http.Error(w, "Invalid or already used activation code", http.StatusUnauthorized)
			return
		}
	}

	code := r.URL.Query().Get("code")
	templates.ExecuteTemplate(w, "login.html", code)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value != "some_session_token" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		userInput := r.FormValue("userInput")
		templates.ExecuteTemplate(w, "dashboard.html", userInput)
		return
	}

	templates.ExecuteTemplate(w, "dashboard.html", nil)
}

func validateAndUseActivationCode(code string) bool {
	tx, err := db.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		return false
	}

	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM activation_codes WHERE code = ? AND used = FALSE", code).Scan(&count)
	if err != nil {
		log.Println("Error querying database:", err)
		tx.Rollback()
		return false
	}

	if count == 0 {
		tx.Rollback()
		return false
	}

	_, err = tx.Exec("UPDATE activation_codes SET used = TRUE WHERE code = ?", code)
	if err != nil {
		log.Println("Error updating database:", err)
		tx.Rollback()
		return false
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Error committing transaction:", err)
		return false
	}

	return true
}
