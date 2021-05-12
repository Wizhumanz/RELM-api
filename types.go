package main

import (
	"fmt"
	"reflect"

	"cloud.google.com/go/datastore"
)

// API types

type jsonResponse struct {
	Msg  string `json:"message"`
	Body string `json:"body"`
}

//for unmarshalling JSON to bools
type JSONBool bool

func (bit *JSONBool) UnmarshalJSON(b []byte) error {
	txt := string(b)
	*bit = JSONBool(txt == "1" || txt == "true")
	return nil
}

type loginReq struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	K           *datastore.Key `datastore:"__key__"`
	KEY         string         `json:"KEY,omitempty"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	AccountType string         `json:"type"`
	Password    string         `json:"password"`
	PhoneNumber string         `json:"phone"`
	AgencyID    string         `json:"agencyID"`
}

type TwilioReq struct {
	OwnerNumber string `json:"owner"`
}

func (l User) String() string {
	r := ""
	v := reflect.ValueOf(l)
	typeOfL := v.Type()

	for i := 0; i < v.NumField(); i++ {
		r = r + fmt.Sprintf("%s: %v, ", typeOfL.Field(i).Name, v.Field(i).Interface())
	}
	return r
}

type Agency struct {
	KEY  string         `json:"KEY,omitempty"`
	K    *datastore.Key `datastore:"__key__"`
	Name string         `json:"name"`
	URL  string         `json:"URL"`
}

type Listing struct {
	K             *datastore.Key `datastore:"__key__"`
	KEY           string         `json:"KEY,omitempty"`
	AggregateID   int            `json:"AggregateID,string"`
	Agency        string         `json:"agency"`
	User          string         `json:"user"`
	OwnerName     string         `json:"ownerName"`
	Owner         string         `json:"owner"`
	OwnerPhone    string         `json:"ownerPhone"`
	Name          string         `json:"name"` // immutable once created, used for queries
	Address       string         `json:"address"`
	Postcode      string         `json:"postcode"`
	Area          string         `json:"area"`
	Price         int            `json:"price,string"`
	PropertyType  int            `json:"propertyType,string"` // 0 = landed, 1 = apartment
	ListingType   int            `json:"listingType,string"`  // 0 = for rent, 1 = for sale
	AvailableDate string         `json:"availableDate"`
	IsPublic      bool           `json:"isPublic,string"`
	IsCompleted   bool           `json:"isCompleted,string"`
	IsPending     bool           `json:"isPending,string"`
	Imgs          []string       `json:"imgs"`
	Timestamp     string         `json:"Timestamp,omitempty"`
	Sqft          int            `json:"sqft,string"`
	Remarks       string         `json:"remarks"`
}

func (l Listing) String() string {
	r := ""
	v := reflect.ValueOf(l)
	typeOfL := v.Type()

	for i := 0; i < v.NumField(); i++ {
		r = r + fmt.Sprintf("%s: %v, ", typeOfL.Field(i).Name, v.Field(i).Interface())
	}
	return r
}
