package main

import "net/http"

func HandleHomepage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "pages/index.html")
}
