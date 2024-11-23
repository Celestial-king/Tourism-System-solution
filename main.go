package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/generative-ai-go/genai"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/option"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

type Message struct {
	Message string
}

type Data struct {
	User       User
	Items      []Item
	Item       Item
	Booking    Booking
	Message    Message
	Location   string
	Activities []Activities
}

type Booking struct {
	UserID    uint
	ItemID    uint
	Item      Item
	StartDate string
	EndDate   string
}

type Activities struct {
	ActivityName string `json:"activity_name"`
	Description  string
	UrlLink      string `json:"url_link"`
	ImageLink    string `json:"image_link"`
}

type User struct {
	ID             uint `gorm:"primaryKey;default:auto_random()"`
	FirstName      string
	LastName       string
	Email          string
	ContactNumber  string
	PassportNumber string //optional
	Address        string
	Password       string
	Balance        float64
}

type Item struct {
	ID          uint `gorm:"primaryKey;default:auto_random()"`
	Title       string
	Price       uint
	Description template.HTML
	Location    string
	ImageUrl    string
	MerchantID  uint
}

type Merchant struct {
	ID        uint `gorm:"primaryKey;default:auto_random()"`
	FirstName string
	LastName  string
	AvatarURL string
	Price     uint
	Status    string
	Ratings   uint
}

func main() {

	fmt.Println("Connecting to database..")

	dsn := "root:root@tcp(localhost:3306)/kaya?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		panic(fmt.Sprintf("cannot connect to database", err.Error()))
	}

	fmt.Println("Running Migrations")
	err = db.AutoMigrate(&Item{}, &User{}, &Booking{})

	if err != nil {
		panic(fmt.Sprintf("cannot connect to database", err.Error()))
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/suggest", func(w http.ResponseWriter, r *http.Request) {
		location := r.FormValue("location")
		//initialize gemini
		ctx := context.Background()
		var user User

		// Access your API key as an environment variable
		client, err := genai.NewClient(ctx, option.WithAPIKey(""))
		if err != nil {
			log.Fatal(err)
		}

		defer client.Close()

		model := client.GenerativeModel("gemini-1.5-flash")
		model.ResponseMIMEType = "application/json"
		prompt := fmt.Sprintf(`suggest activities that we can do in %v using this JSON schema:Itinerary = {'destination': string, 'time': string, 'activity_name': string, 'description': string, 'url_link': string} Return: Array<Itinerary>`, location)
		resp, err := model.GenerateContent(ctx, genai.Text(prompt))

		var activities []Activities
		content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

		err = json.Unmarshal([]byte(content), &activities)

		if err != nil {
			panic(err.Error())
		}

		t, err := template.ParseFiles("./pages/suggestion.html")

		if err != nil {
			panic(err)
		}

		cookie, err := r.Cookie("login_session")
		if err == nil {
			userID := cookie.Value

			id, _ := strconv.Atoi(userID)
			if id != 0 {
				user = User{
					ID: uint(id),
				}
				db.First(&user)
			}
		}

		t.Execute(w, Data{
			Activities: activities,
			User:       user,
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		var items []Item
		var user User
		db.Limit(10).Find(&items)

		t, err := template.ParseFiles("./pages/index.html")

		if err != nil {
			panic(err)
		}

		cookie, err := r.Cookie("login_session")
		if err == nil {
			userID := cookie.Value

			id, _ := strconv.Atoi(userID)
			if id != 0 {
				user = User{
					ID: uint(id),
				}
				db.First(&user)
			}
		}

		t.Execute(w, &Data{
			Items: items,
			User:  user,
		})
	})

	r.Get("/merchant", func(w http.ResponseWriter, r *http.Request) {
		merchant := Merchant{
			FirstName: "Trisha",
		}

		t, err := template.ParseFiles("./pages/merchant-dashboard.html")

		if err != nil {
			panic(err)
		}

		t.Execute(w, merchant)
	})

	r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("./pages/login.html")

		if err != nil {
			panic(err)
		}

		t.Execute(w, &Data{})

		return
	})

	r.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		cookie := &http.Cookie{
			Name:   "login_session",
			MaxAge: -1,
		}
		http.SetCookie(w, cookie)

		t, err := template.ParseFiles("./pages/login.html")

		if err != nil {
			panic(err)
		}

		t.Execute(w, &Data{})

		return
	})

	r.Get("/signup", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/sign-up.html")

		return
	})

	r.Get("/activities/{id}", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("./pages/product_details.html")

		if err != nil {
			panic(err)
		}

		id := chi.URLParam(r, "id")
		numericID, _ := strconv.Atoi(id)

		var item Item
		item.ID = uint(numericID)

		db.First(&item)
		item.Description = item.Description
		var user User

		cookie, err := r.Cookie("login_session")
		if err == nil {
			userID := cookie.Value

			id, _ := strconv.Atoi(userID)
			if id != 0 {
				user = User{
					ID: uint(id),
				}
				db.First(&user)
			}
		}

		t.Execute(w, &Data{
			Item: item,
			User: user,
		})
		return
	})

	r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
		location := r.URL.Query().Get("location")
		var items []Item
		var user User

		if location == "" {
			db.Limit(10).Find(&items)
		} else {
			db.Where("location=?", location).Find(&items)
		}

		cookie, err := r.Cookie("login_session")
		if err == nil {
			userID := cookie.Value

			id, _ := strconv.Atoi(userID)
			if id != 0 {
				user = User{
					ID: uint(id),
				}
				db.First(&user)
			}
		}

		t, err := template.ParseFiles("./pages/results.html")

		if err != nil {
			panic(err.Error())
		}

		t.Execute(w, &Data{
			Items:    items,
			User:     user,
			Location: location,
		})
	})

	r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
		password := r.FormValue("password")
		bytePassword, _ := bcrypt.GenerateFromPassword([]byte(password), 14)

		user := User{
			FirstName:     r.FormValue("firstName"),
			LastName:      r.FormValue("lastName"),
			Email:         r.FormValue("email"),
			ContactNumber: r.FormValue("contactNumber"),
			Password:      string(bytePassword),
		}

		db.Create(&user)

		cookie := http.Cookie{
			Name:   "login_session",
			Value:  fmt.Sprintf("%v", user.ID),
			Path:   "/",
			MaxAge: 3600,
		}

		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		var user User
		db.Where("email = ?", r.FormValue("email")).First(&user)

		if user.Email == "" {
			t, err := template.ParseFiles("./pages/login.html")

			if err != nil {
				panic(err)
			}

			t.Execute(w, Data{
				Message: Message{
					Message: "Invalid username / password",
				},
			})

			return
		}

		//check password
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(r.FormValue("password")))

		if err != nil {
			t, err := template.ParseFiles("./pages/login.html")

			if err != nil {
				panic(err)
			}

			t.Execute(w, Data{
				Message: Message{
					Message: "Invalid Username / Password",
				},
			})
		}

		cookie := http.Cookie{
			Name:   "login_session",
			Value:  fmt.Sprintf("%v", user.ID),
			Path:   "/",
			MaxAge: 3600,
		}

		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	fs := http.FileServer(http.Dir("./images"))
	r.Handle("/images/*", http.StripPrefix("/images/", fs))

	r.Post("/bookings", func(w http.ResponseWriter, r *http.Request) {
		var user User
		cookie, err := r.Cookie("login_session")
		if err == nil {
			userID := cookie.Value

			id, _ := strconv.Atoi(userID)
			if id != 0 {
				user = User{
					ID: uint(id),
				}
				db.First(&user)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
		}

		startDate := r.FormValue("startDate")
		endDate := r.FormValue("endDate")

		item := r.FormValue("itemID")

		start, err := time.Parse("2006-01-02", startDate)
		end, err := time.Parse("2006-01-02", endDate)
		numericItem, _ := strconv.Atoi(item)

		bookedItem := Item{
			ID: uint(numericItem),
		}

		db.First(&bookedItem)

		difference := math.Abs(start.Sub(end).Hours() / 24)

		cost := (float64(bookedItem.Price) * difference) + 1

		user.Balance = user.Balance - cost
		db.Save(&user)

		itemID, err := strconv.Atoi(item)

		//fetch item to see its price
		booking := Booking{
			StartDate: startDate,
			EndDate:   endDate,
			ItemID:    uint(itemID),
			UserID:    user.ID,
		}

		db.Create(&booking)

		t, err := template.ParseFiles("./pages/bookings.html")

		if err != nil {
			panic(err)
		}

		t.Execute(w, Data{
			User:    user,
			Item:    bookedItem,
			Booking: booking,
		})
	})

	r.Get("/hekhek", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("./pages/account.html")

		if err != nil {
			panic(err.Error())
		}

		t.Execute(w, nil)
	})

	fmt.Println("Server running at localhost:3000")
	err = http.ListenAndServe(":3000", r)

	if err != nil {
		panic("Error starting the server")
	}
}
