package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"html/template"
	"net/http"
)

type User struct {
	ID             uint `gorm:"primaryKey;default:auto_random()"`
	FirstName      string
	LastName       string
	Email          string
	ContactNumber  string
	PassportNumber string //optional
	Address        string
}

type Item struct {
	ID          uint `gorm:"primaryKey;default:auto_random()"`
	Code        string
	Price       uint
	Description string
	Location    string
	ImageUrl    string
}

func main() {
	fmt.Println("Connecting to database..")

	dsn := "root:root@tcp(localhost:3306)/kaya?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		panic(fmt.Sprintf("cannot connect to database", err.Error()))
	}

	fmt.Println("Running Migrations")
	err = db.AutoMigrate(&Item{}, &User{})

	if err != nil {
		panic(fmt.Sprintf("cannot connect to database", err.Error()))
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		var items Item
		db.Limit(10).Find(&items)

		t, err := template.ParseFiles("./pages/index.html")

		if err != nil {
			panic(err)
		}

		t.Execute(w, items)
	})

	r.Get("/verify", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/results.html")
	})

	fmt.Println("Server running at localhost:3000")
	err = http.ListenAndServe(":3000", r)

	if err != nil {
		panic("Error starting the server")
	}
}
