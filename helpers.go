package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"encoding/csv"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"golang.org/x/crypto/bcrypt"
)

func setupCORS(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Content-Type", "text/html; charset=utf-8")
	//(*w).Header().Set("Access-Control-Expose-Headers", "Authorization")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, auth, Cache-Control, Pragma, Expires")
}

// helper funcs

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 2)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func authenticateUser(req loginReq) (bool, User) {
	// get user with id/email
	var userWithEmail User
	var query *datastore.Query
	if req.Email != "" {
		query = datastore.NewQuery("User").
			Filter("Email =", req.Email)
	} else if req.ID != "" {
		i, _ := strconv.Atoi(req.ID)
		key := datastore.IDKey("User", int64(i), nil)
		query = datastore.NewQuery("User").
			Filter("__key__ =", key)
	} else {
		return false, User{}
	}

	t := client.Run(ctx, query)
	_, error := t.Next(&userWithEmail)
	if error != nil {
		fmt.Println(error.Error())
	}
	// check password hash and return
	return CheckPasswordHash(req.Password, userWithEmail.Password), userWithEmail
}

func deleteElement(sli []Listing, del Listing) []Listing {
	var rSli []Listing
	for _, e := range sli {
		if e.KEY != del.KEY {
			rSli = append(rSli, e)
		}
	}
	return rSli
}

type checkerFunc func(Listing) bool

func GetIndex(s []Listing, chk checkerFunc) int {
	for i, li := range s {
		if chk(li) {
			return i
		}
	}
	return 0
}

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}

func createNewListingExcel(w http.ResponseWriter, r *http.Request, myListing Listing) {
	// add owner information
	var newUser User
	newUser.Name = myListing.OwnerName
	newUser.Email = myListing.Owner
	newUser.PhoneNumber = myListing.OwnerPhone
	newUser.AccountType = "owner"
	// set password hash
	newUser.Password, _ = HashPassword(newUser.Password)
	newUser.AgencyID = myListing.Agency

	// create new user in DB
	kind := "User"
	name := time.Now().Format("2006-01-02_15:04:05_-0700")
	newUserKey := datastore.NameKey(kind, name, nil)

	if _, err := client.Put(ctx, newUserKey, &newUser); err != nil {
		log.Fatalf("Failed to save User: %v", err)
	}
	fmt.Println("createNewListingExcel + " + myListing.Name)

	addListing(w, r, false, myListing, true, true) //empty Listing struct passed just for compiler
}
