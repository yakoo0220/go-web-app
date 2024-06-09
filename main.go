package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	templates = template.Must(template.ParseFiles("templates/login.html", "templates/dashboard.html"))
	db        *sql.DB
)

func main() {
	var err error
	db, err = sql.Open("mysql", "root:7ea5837b98d523dc@tcp(127.0.0.1:3306)/go_web_app")
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
		if validateAndMarkLogin(activationCode) {
			http.SetCookie(w, &http.Cookie{
				Name:  "session_token",
				Value: activationCode, // Store activation code in the session token
			})
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		} else {
			templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
				"Error": "Invalid or already used activation code",
				"Code":  activationCode,
			})
			return
		}
	}

	code := r.URL.Query().Get("code")
	templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
		"Code": code,
	})
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	activationCode := cookie.Value

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if !isValidEmail(username) || !isValidPassword(password) {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		if saveUserCredentialsAndMarkUsed(activationCode, username, password) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else {
			http.Error(w, "Failed to save user credentials", http.StatusInternalServerError)
			return
		}
	}

	templates.ExecuteTemplate(w, "dashboard.html", nil)
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	return re.MatchString(email)
}

func isValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[@$!%*?&]`).MatchString(password)
	return hasUpper && hasLower && hasNumber && hasSpecial
}

func saveUserCredentialsAndMarkUsed(activationCode, username, password string) bool {
	tx, err := db.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		return false
	}

	now := time.Now()
	_, err = tx.Exec("UPDATE activation_codes SET email = ?, password = ?, used = TRUE, registration_time = ? WHERE code = ?", username, password, now, activationCode)
	if err != nil {
		log.Println("Error updating user credentials:", err)
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

func validateAndMarkLogin(code string) bool {
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

	now := time.Now()
	_, err = tx.Exec("UPDATE activation_codes SET islogin = TRUE, login_time = ? WHERE code = ?", now, code)
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
