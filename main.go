package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/gorilla/mux"
)

var googleProjectID = "myika-relm"
var client *datastore.Client
var ctx context.Context

func main() {
	//init
	ctx = context.Background()
	var err error
	client, err = datastore.NewClient(ctx, googleProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET", "OPTIONS").Path("/").HandlerFunc(indexHandler)
	router.Methods("POST", "OPTIONS").Path("/login").HandlerFunc(loginHandler)
	router.Methods("POST", "OPTIONS").Path("/user").HandlerFunc(createNewUserHandler)
	router.Methods("GET", "OPTIONS").Path("/owner").HandlerFunc(getUserHandler)
	router.Methods("GET", "OPTIONS").Path("/listings").HandlerFunc(getAllListingsHandler)
	router.Methods("GET", "OPTIONS").Path("/agency").HandlerFunc(getAllAgency)
	router.Methods("POST", "OPTIONS").Path("/listing").HandlerFunc(createNewListingHandler)
	router.Methods("PUT", "OPTIONS").Path("/listing/{id}").HandlerFunc(updateListingHandler)
	router.Methods("POST", "OPTIONS").Path("/twilio").HandlerFunc(getOwnerNumberHandler)
	router.Methods("POST", "OPTIONS").Path("/upload").HandlerFunc(createNewListingsExcel)

	port := os.Getenv("PORT")
	fmt.Println("relm-api listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
