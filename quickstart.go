package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// This is based off Google tutorial

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	// Prints the email, last name and tags of residents in westport spreadsheet:
	// spreadsheet id is in the URL, e.g. https://docs.google.com/spreadsheets/d/1wiQ8LIaUXnkpCfCdFEvSycmY590twbTuQCDYahVs99Q/edit#gid=1841356976
	// First is HepTestOne
	spreadsheetId := "1wiQ8LIaUXnkpCfCdFEvSycmY590twbTuQCDYahVs99Q"
	// Second is Directory2019_workingCopy under westporter1 account
	// spreadsheetId := "1Z-De3hCJwKRBWrTEB97QG7z7aVGysIWUJzYDIDzJMgk"
	readRange := "Entire data base!A2:D5"
	response, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	type Tag struct {
		Name string `json:"name,omitempty"`
	}

	type AudienceMember struct {
		HashedEmail string `json:"id,omitempty"`
		Email       string `json:"email_address,omitempty"`
		Tags        []Tag  `json:"tags,omitempty"`
	}

	mailchimpUpload := []AudienceMember{}

	updateDate := fmt.Sprintf("%s %v %v", time.Now().Month().String(), time.Now().Day(), time.Now().Year())

	if len(response.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		for _, row := range response.Values {
			if row[0] == "" {
				continue
			}
			member := AudienceMember{}
			member.Email = row[0].(string)
			emailHashing := md5.Sum([]byte(member.Email))
			member.HashedEmail = fmt.Sprintf("%x", emailHashing)

			paidStatus := "Unpaid"
			if strings.Contains(row[3].(string), "paid") {
				paidStatus = "Paid"
			}

			member.Tags = []Tag{
				{Name: paidStatus},
				{Name: updateDate},
			}
			mailchimpUpload = append(mailchimpUpload, member)
		}
	}
	// fmt.Printf("%+v", mailchimpUpload)
	payload, err := json.Marshal(mailchimpUpload)
	if err != nil {
		fmt.Println("error:", err)
	}
	os.Stdout.Write(payload)
}
