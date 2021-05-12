package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
)

// route handlers

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	userEmail := r.URL.Query().Get("owner")
	if userEmail == "" {
		//Handle error.
	}

	var userWithEmail User
	query := datastore.NewQuery("User").
		Filter("Email =", userEmail)
	t := client.Run(ctx, query)
	_, error := t.Next(&userWithEmail)
	if error != nil {
		// Handle error.
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userWithEmail)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	var data jsonResponse
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "GET" {
		data = jsonResponse{Msg: "Only GET Allowed", Body: "This endpoint only accepts GET requests."}
		w.WriteHeader(http.StatusUnauthorized)
		return
	} else {
		data = jsonResponse{Msg: "RELM API", Body: "Ready"}
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(data)
	// w.Write([]byte(`{"msg": "привет сука"}`))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	var newLoginReq loginReq
	// decode data
	err := json.NewDecoder(r.Body).Decode(&newLoginReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var data jsonResponse

	loginSuccess, theUser := authenticateUser(newLoginReq)

	query := datastore.NewQuery("User").Filter("Email =", newLoginReq.Email)
	t := client.Run(ctx, query)

	var x User
	_, error := t.Next(&x)
	if error != nil {
		// Handle error.
		fmt.Println("Error")
	}

	if err != nil {
		// Handle error.
	}
	//agencyResp := x.AgencyID

	if loginSuccess {
		data = jsonResponse{
			Msg:  fmt.Sprint(theUser.K.ID),
			Body: theUser.AgencyID,
		}
		w.WriteHeader(http.StatusCreated)
	} else {
		data = jsonResponse{
			Msg:  "Authentication failed.",
			Body: newLoginReq.Email,
		}
		w.WriteHeader(http.StatusUnauthorized)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func createNewUserHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	var newUser User
	// decode data
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// set password hash
	newUser.Password, _ = HashPassword(newUser.Password)

	// create new listing in DB
	kind := "User"
	newUserKey := datastore.IncompleteKey(kind, nil)

	if _, err := client.Put(ctx, newUserKey, &newUser); err != nil {
		log.Fatalf("Failed to save User: %v", err)
	}

	// return
	data := jsonResponse{
		Msg:  "Set " + newUserKey.String(),
		Body: newUser.String(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func getAllAgency(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	agencyResp := make([]Agency, 0)

	query := datastore.NewQuery("Agency")
	t := client.Run(ctx, query)

	for {
		var x Agency
		key, err := t.Next(&x)
		if key != nil {
			x.KEY = fmt.Sprint(key.ID)
		}
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Handle error.
		}
		agencyResp = append(agencyResp, x)
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	json.NewEncoder(w).Encode(agencyResp)
}

func getAllListingsHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	listingsResp := make([]Listing, 0)

	authReq := loginReq{
		ID:       r.URL.Query()["user"][0],
		Password: r.Header.Get("auth"),
	}

	//only need to authenticate if not fetching public listings
	loginSuccess, _ := authenticateUser(authReq)
	if len(r.URL.Query()["isPublic"]) == 0 && !loginSuccess {
		data := jsonResponse{Msg: "Authorization Invalid", Body: "Go away."}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(data)
		return
	}

	var query *datastore.Query
	agencyIDParam := r.URL.Query()["agency"][0]
	var isPublicParam = true //default
	if len(r.URL.Query()["isPublic"]) > 0 {
		//extract correct isPublic param
		isPublicQueryStr := r.URL.Query()["isPublic"][0]
		if isPublicQueryStr == "true" {
			isPublicParam = true
		} else if isPublicQueryStr == "false" {
			isPublicParam = false
		}

		query = datastore.NewQuery("Listing").
			Filter("Agency =", agencyIDParam).
			Filter("IsPublic =", isPublicParam)
	} else {
		query = datastore.NewQuery("Listing").
			Filter("Agency =", agencyIDParam)
	}

	//run query, decode listings to obj and store in slice
	t := client.Run(ctx, query)
	for {
		var x Listing
		_, err := t.Next(&x)

		if x.K != nil {
			x.KEY = fmt.Sprint(x.K.ID)
		}
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Handle error.
		}
		listingsResp = append(listingsResp, x)
	}

	//if no listings, return empty array
	if len(listingsResp) == 0 {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listingsResp)
		return
	}

	//sum up event sourcing for listings
	var existingListingsArr []Listing
	for _, li := range listingsResp {
		//find listing in existing listings array
		var exListing Listing
		for i := range existingListingsArr {
			if existingListingsArr[i].AggregateID == li.AggregateID {
				exListing = existingListingsArr[i]
			}
		}
		//if listings already exists, remove one
		if exListing.AggregateID != 0 {
			//compare date keys
			layout := "2006-01-02_15:04:05_-0700"
			existingTime, _ := time.Parse(layout, exListing.Timestamp)
			currentLITime, _ := time.Parse(layout, li.Timestamp)
			//if existing is older, remove and add newer current listing; otherwise, do nothing
			if existingTime.Before(currentLITime) {
				//rm existing listing
				existingListingsArr = deleteElement(existingListingsArr, exListing)
				//append current listing
				existingListingsArr = append(existingListingsArr, li)
			}
		} else {
			existingListingsArr = append(existingListingsArr, li)
		}
	}
	listingsResp = existingListingsArr

	//only get images for some listings
	var imgFetchListings []Listing
	var startAtID string //lazy loading
	if len(listingsResp) > 0 && len(r.URL.Query()["startID"]) > 0 {
		//start fetching images from last ID passed by client
		startAtID = r.URL.Query()["startID"][0]
		indexOfStartID := GetIndex(listingsResp, func(l Listing) bool {
			return l.KEY == startAtID
		})
		imgFetchListings = listingsResp[indexOfStartID:]
	} else if len(listingsResp) > 0 {
		respLen := len(listingsResp)
		if respLen > 4 {
			respLen = 4
			imgFetchListings = listingsResp[:respLen]
		} else {
			imgFetchListings = listingsResp
		}
	}

	//cloud storage connection config
	storageClient, _ := storage.NewClient(ctx)
	defer storageClient.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	//get images from cloud storage buckets
	var imgFilledListings []Listing = []Listing{}
	for _, li := range imgFetchListings {
		imgArr := li.Imgs
		if len(imgArr) <= 0 {
			continue
		}
		bkt := li.Imgs[0]

		//list all objects in bucket
		var objNames []string = []string{}
		it := storageClient.Bucket(bkt).Objects(ctx, nil)
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				// return fmt.Errorf("Bucket(%q).Objects: %v", bkt, err)
			}
			if attrs != nil {
				objNames = append(objNames, attrs.Name)
			} else {
				break
			}
		}

		//download first img, set as new img property for listing (decoded on client side)
		var imgStrs []string
		// if len(objNames) > 0 {
		// 	obj := objNames[0]
		// 	rc, _ := storageClient.Bucket(bkt).Object(obj).NewReader(ctx)
		// 	defer rc.Close()

		// 	imgByteArr, _ := ioutil.ReadAll(rc)
		// 	imgStrs = append(imgStrs, base64.StdEncoding.EncodeToString(imgByteArr))
		// }
		for _, s := range objNames {
			obj := s
			rc, _ := storageClient.Bucket(bkt).Object(obj).NewReader(ctx)
			defer rc.Close()

			imgByteArr, _ := ioutil.ReadAll(rc)
			imgStrs = append(imgStrs, base64.StdEncoding.EncodeToString(imgByteArr))
		}

		// old GET all
		// for _, obj := range objNames {
		// 	rc, err := client.Bucket(bkt).Object(obj).NewReader(ctx)
		// 	if err != nil {
		// 		// return nil, fmt.Errorf("Object(%q).NewReader: %v", obj, err)
		// 	}
		// 	defer rc.Close()

		// 	imgByteArr, err := ioutil.ReadAll(rc)
		// 	if err != nil {
		// 		// return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
		// 	}
		// 	imgStrs = append(imgStrs, base64.StdEncoding.EncodeToString(imgByteArr))
		// }

		li.Imgs = imgStrs
		imgFilledListings = append(imgFilledListings, li)
	}
	//build resp array
	var finalResp []Listing
	for _, li := range listingsResp {
		//determine if current listing just got filled with imgs
		filledListing := Listing{}
		for _, f := range imgFilledListings {
			if f.KEY == li.KEY {
				filledListing = f
			}
		}
		//return filled img listings
		if filledListing.Name != "" {
			finalResp = append(finalResp, filledListing)
		} else {
			finalResp = append(finalResp, li)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResp)
}

// almost identical logic with create and update (event sourcing)
func addListing(w http.ResponseWriter, r *http.Request, isPutReq bool, listingToUpdate Listing, doNotDecode bool, isExcel bool) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}
	var listingToUse Listing

	// decode data
	if !doNotDecode {
		err := json.NewDecoder(r.Body).Decode(&listingToUse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		listingToUse = listingToUpdate
	}

	if !isExcel {
		authReq := loginReq{
			ID:       listingToUse.User,
			Password: r.Header.Get("auth"),
		}
		// for PUT req, userEmail already authenticated outside this function
		loginSuccess, _ := authenticateUser(authReq)
		if !isPutReq && !loginSuccess {
			data := jsonResponse{Msg: "Authorization Invalid", Body: "Go away."}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(data)
			return
		}
	}
	// if updating listing, don't allow Name change
	// if isPutReq && (listingToUse.Name != "") {
	// 	data := jsonResponse{Msg: "Name property of Listing is immutable.", Body: "Do not pass Name property in request body."}
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	json.NewEncoder(w).Encode(data)
	// 	return
	// }
	// if updating, name field not passed in JSON body, so must fill
	if isPutReq {
		listingToUse.AggregateID = listingToUpdate.AggregateID
	} else {
		// else increment aggregate ID
		var x Listing
		//get highest aggregate ID
		query := datastore.NewQuery("Listing").
			Project("AggregateID").
			Order("-AggregateID")
		t := client.Run(ctx, query)
		_, error := t.Next(&x)
		if error != nil {
			// Handle error.
			fmt.Println("Error")
		}
		listingToUse.AggregateID = x.AggregateID + 1
	}

	//set timestamp
	listingToUse.Timestamp = time.Now().Format("2006-01-02_15:04:05_-0700")

	//var newListingName string
	// TODO: fill empty PUT listing fields
	if !isExcel {
		//must have images to POST new listing
		if !isPutReq && len(listingToUse.Imgs) <= 0 {
			data := jsonResponse{Msg: "No images found in body.", Body: "At least one image must be included to create a new listing."}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(data)
			return
		}
		//save images in new bucket on POST only
		ctx := context.Background()
		//use listing ID as bucket name
		//newListingName = time.Now().Format("2006-01-02_15:04:05_-0700")
		if !isPutReq {
			clientStorage, err := storage.NewClient(ctx)
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			//format for proper bucket name
			//bucketName := strings.ReplaceAll(newListingName, ":", "-") //url.QueryEscape(newListing.UserID + "." + newListing.Name)
			//bucketName = strings.ReplaceAll(bucketName, "+", "plus")
			//bucketName := listingToUse.AggregateID
			bucketName := "agg_" + strconv.Itoa(listingToUse.AggregateID)
			bucket := clientStorage.Bucket(bucketName)
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()
			if err := bucket.Create(ctx, googleProjectID, nil); err != nil {
				log.Fatalf("Failed to create bucket: %v", err)
			}

			for j, strImg := range listingToUse.Imgs {
				//fmt.Println(strImg)
				//convert image from base64 string to JPEG
				i := strings.Index(strImg, ",")
				if i < 0 {
					fmt.Println("img in body no comma")
				}

				//store img in new bucket
				dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(strImg[i+1:])) // pass reader to NewDecoder
				// Upload an object with storage.Writer.
				wc := clientStorage.Bucket(bucketName).Object(fmt.Sprintf("%d", j)).NewWriter(ctx)
				if _, err = io.Copy(wc, dec); err != nil {
					fmt.Printf("io.Copy: %v", err)
				}
				if err := wc.Close(); err != nil {
					fmt.Printf("Writer.Close: %v", err)
				}
			}
			listingToUse.Imgs = []string{bucketName} //just store bucket name, objects retrieved on getListing
		} else {
			listingToUse.Imgs = listingToUpdate.Imgs
		}
	} else {
		//newListingName = time.Now().Format("2006-01-02_15:04:05_-0700")
	}

	// create new listing in DB
	kind := "Listing"
	newListingKey := datastore.IncompleteKey(kind, nil)

	if _, err := client.Put(ctx, newListingKey, &listingToUse); err != nil {
		log.Fatalf("Failed to save Listing: %v", err)
	}

	// return
	data := jsonResponse{
		Msg:  "Added listing",
		Body: listingToUse.String(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func updateListingHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	//check if listing already exists to update
	putID, unescapeErr := url.QueryUnescape(mux.Vars(r)["id"]) //is actually Listing.Name, not __key__ in Datastore
	if unescapeErr != nil {
		data := jsonResponse{Msg: "Listing ID Parse Error", Body: unescapeErr.Error()}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(data)
		return
	}
	listingsResp := make([]Listing, 0)

	//auth
	authReq := loginReq{
		ID:       r.URL.Query()["user"][0],
		Password: r.Header.Get("auth"),
	}
	loginSuccess, _ := authenticateUser(authReq)
	if !loginSuccess {
		data := jsonResponse{Msg: "Authorization Invalid", Body: "Go away."}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(data)
		return
	}
	int, _ := strconv.Atoi(putID)

	//get listing with ID
	query := datastore.NewQuery("Listing").
		Filter("AggregateID =", int)

	t := client.Run(ctx, query)
	for {
		var x Listing
		_, err := t.Next(&x)

		if err == iterator.Done {
			break
		}
		if err != nil {
			// Handle error.
		}
		if x.K != nil {
			x.KEY = fmt.Sprint(x.K.ID)
		}

		//event sourcing (pick latest snapshot)
		if len(listingsResp) == 0 {
			listingsResp = append(listingsResp, x)
		} else {
			//find bot in existing array
			var exListing Listing
			for _, b := range listingsResp {
				if b.AggregateID == x.AggregateID {
					exListing = b
				}
			}

			//if bot exists, append row/entry with the latest timestamp
			if exListing.AggregateID != 0 || exListing.Timestamp != "" {

				//compare timestamps
				layout := "2006-01-02_15:04:05_-0700"
				existingBotTime, _ := time.Parse(layout, exListing.Timestamp)
				newBotTime, _ := time.Parse(layout, x.Timestamp)
				//if existing is older, remove it and add newer current listing; otherwise, do nothing
				if existingBotTime.Before(newBotTime) {

					//rm existing listing
					listingsResp = deleteElement(listingsResp, exListing)
					//append current listing
					listingsResp = append(listingsResp, x)
				}
			} else {
				//otherwise, just append newly decoded (so far unique) bot
				listingsResp = append(listingsResp, x)
			}
		}
	}

	//return if listing to update doesn't exist
	putIDValid := len(listingsResp) > 0 && listingsResp[0].Address != ""
	if !putIDValid {
		data := jsonResponse{Msg: "Listing ID Invalid", Body: "Listing with provided Name does not exist."}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(data)
		return
	}

	addListing(w, r, true, listingsResp[len(listingsResp)-1], false, false)
}

func createNewListingHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	var myListing Listing
	// decode data
	err := json.NewDecoder(r.Body).Decode(&myListing)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//fmt.Println(myListing.String())

	var userWithEmail User
	query := datastore.NewQuery("User")

	t := client.Run(ctx, query)
	_, error := t.Next(&userWithEmail)
	if error != nil {
		// Handle error.
		fmt.Println("Error")
	}

	if userWithEmail.Email == myListing.Owner && userWithEmail.Name == myListing.OwnerName && userWithEmail.PhoneNumber == myListing.OwnerPhone {
		data := jsonResponse{Msg: "Owner already exists", Body: "Input new owner"}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(data)
		return
	} else if userWithEmail.Email == myListing.Owner || userWithEmail.PhoneNumber == myListing.OwnerPhone {
		data := jsonResponse{Msg: "Owner already exists", Body: "Email or phone number already in use"}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(data)
		return
	}

	// add owner information
	var newUser User
	newUser.Name = myListing.OwnerName
	newUser.Email = myListing.Owner
	newUser.PhoneNumber = myListing.OwnerPhone
	newUser.AccountType = "owner"
	// set password hash
	newUser.Password, _ = HashPassword(newUser.Password)

	// create new user in DB
	kind := "User"
	newUserKey := datastore.IncompleteKey(kind, nil)

	if _, err := client.Put(ctx, newUserKey, &newUser); err != nil {
		log.Fatalf("Failed to save User: %v", err)
	}

	addListing(w, r, false, myListing, true, false) //empty Listing struct passed just for compiler

	// return
	data := jsonResponse{
		Msg:  "Set " + myListing.Name,
		Body: myListing.String(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func getOwnerNumberHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	accountSid := "ACa59451c872071e8037cf59811057fd21"
	authToken := "3b6a2f39bb05f5214283ef7bd6db973f"
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"
	var twilioReq TwilioReq

	err := json.NewDecoder(r.Body).Decode(&twilioReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v := url.Values{}
	v.Set("To", twilioReq.OwnerNumber)
	v.Set("From", "+15076160092")
	v.Set("Body", "Brooklyn's in the house!")
	rb := *strings.NewReader(v.Encode())

	// Create Client
	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &rb)
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := client.Do(req)
	fmt.Println(resp.Status)
}

func createNewListingsExcel(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	agencyID := r.URL.Query().Get("agencyID")
	user := r.URL.Query().Get("user")
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, _, err := r.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile(os.TempDir(), "upload-*.csv")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	//Read csv file from the given file path
	records := readCsvFile(tempFile.Name())

	var excelListing Listing

	for i := 1; i < len(records); i++ {
		excelListing.Name = records[i][0]
		excelListing.Address = records[i][0] + " " + records[i][1]
		excelListing.Postcode = records[i][2]
		excelListing.Area = records[i][3]
		excelListing.OwnerName = records[i][4]
		excelListing.OwnerPhone = records[i][5]
		excelListing.Owner = records[i][6]
		s, _ := strconv.Atoi(records[i][7])
		excelListing.Price = s
		if strings.Contains(strings.ToLower(records[i][8]), "apartment") {
			excelListing.PropertyType = 1
		} else if strings.Contains(strings.ToLower(records[i][8]), "landed") {
			excelListing.PropertyType = 0
		} else {
			excelListing.PropertyType = 2
		}

		if strings.Contains(strings.ToLower(records[i][9]), "rent") {
			excelListing.ListingType = 0
		} else if strings.Contains(strings.ToLower(records[i][9]), "sale") {
			excelListing.ListingType = 1
		} else {
			excelListing.ListingType = 2
		}

		excelListing.AvailableDate = time.Now().Format("2006-01-02")
		excelListing.Timestamp = time.Now().Format("2006-01-02_15:04:05_-0700")
		excelListing.Agency = agencyID
		excelListing.User = user

		excelListing.IsPublic = false
		excelListing.IsCompleted = false
		excelListing.IsPending = false

		f, _ := strconv.Atoi(records[i][14])
		excelListing.Sqft = f
		excelListing.Remarks = records[i][15]

		createNewListingExcel(w, r, excelListing)
	}
	data := jsonResponse{
		Msg:  "Working",
		Body: agencyID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}
