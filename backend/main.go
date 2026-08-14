package main

import (
"database/sql"
"encoding/json"
"log"
"net/http"
"os"

"github.com/gorilla/mux"
_ "github.com/lib/pq"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// Connect to database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	// Create the table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	// Create router
	router := mux.NewRouter()

	// Enable CORS
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			next.ServeHTTP(w, r)
		})
	})

	router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
	})

	router.HandleFunc("/users", getUsers(db)).Methods("GET")
	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    	w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil{
			log.Println("failed to write health response:", err)
		}
	})
	// Ready endpoint
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
    	w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("READY")); err != nil{
			log.Println("failed to write ready response:", err)
		}
	})
	router.HandleFunc("/users/{id}", getUser(db)).Methods("GET")
	router.HandleFunc("/users", createUser(db)).Methods("POST")
	router.HandleFunc("/users/{id}", updateUser(db)).Methods("PUT")
	router.HandleFunc("/users/{id}", deleteUser(db)).Methods("DELETE")

	// Start server
	log.Fatal(http.ListenAndServe(":8000", jsonContentTypeMiddleware(router)))
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Get all users
func getUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT * FROM users")
		if err != nil {
			log.Printf("Error querying users: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		defer func() {
			if err := rows.Close(); err != nil {
				log.Printf("error closing rows: %v", err)
			}
		}()

		users := []User{}

		for rows.Next() {
			var u User

			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				log.Printf("Error scanning user: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			users = append(users, u)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Error iterating over users: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(users); err != nil {
			log.Printf("Error encoding users: %v", err)
		}
	}
}

// Get user by ID
func getUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var u User

		err := db.QueryRow(
		"SELECT * FROM users WHERE id = $1",
		id,
		).Scan(&u.ID, &u.Name, &u.Email)

		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(u); err != nil {
			log.Printf("Error encoding user: %v", err)
		}
	}
}

// Create user
func createUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u User

		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err := db.QueryRow(
		"INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
		u.Name,
		u.Email,
		).Scan(&u.ID)

		if err != nil {
			log.Printf("Error creating user: %v", err)
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(u); err != nil {
			log.Printf("Error encoding user: %v", err)
		}
	}
}

// Update user
func updateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u User

		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		_, err := db.Exec(
		"UPDATE users SET name = $1, email = $2 WHERE id = $3",
		u.Name,
		u.Email,
		id,
		)

		if err != nil {
			log.Printf("Error updating user: %v", err)
			http.Error(w, "Error updating user", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(u); err != nil {
			log.Printf("Error encoding user: %v", err)
		}
	}
}

// Delete user
func deleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Check if the user exists
		var u User

		err := db.QueryRow(
		"SELECT * FROM users WHERE id = $1",
		id,
		).Scan(&u.ID, &u.Name, &u.Email)

		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)

				if err := json.NewEncoder(w).Encode(
				map[string]string{"error": "User not found"},
				); err != nil {
					log.Printf("Error encoding response: %v", err)
				}

				return
			}

			w.WriteHeader(http.StatusInternalServerError)

			if err := json.NewEncoder(w).Encode(
			map[string]string{"error": "Internal server error"},
			); err != nil {
				log.Printf("Error encoding response: %v", err)
			}

			return
		}

		// User found, delete
		_, err = db.Exec("DELETE FROM users WHERE id = $1", id)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			if err := json.NewEncoder(w).Encode(
			map[string]string{"error": "Error deleting user"},
			); err != nil {
				log.Printf("Error encoding response: %v", err)
			}

			return
		}

		// User successfully deleted
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(
		map[string]string{"message": "User deleted successfully"},
		); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}
}
