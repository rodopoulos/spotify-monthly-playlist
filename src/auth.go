package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const credentialFilePath = ".spotify-token"
const redirectURI = "http://localhost:8080/callback"

var (
	auth = spotify.NewAuthenticator(redirectURI,
		spotify.ScopeUserReadPrivate,
		spotify.ScopePlaylistReadPrivate,
		spotify.ScopePlaylistReadCollaborative,
		spotify.ScopeUserLibraryRead,
		spotify.ScopePlaylistModifyPublic,
		spotify.ScopePlaylistModifyPrivate,
	)
	ch    = make(chan *oauth2.Token)
	state = "abc123"
)

func obtainOAuthToken() *oauth2.Token {
	token := retrieveOAuthTokenFromFile()

	if token == nil {
		log.Print("no token configured. logging in..")
		token = login()
	} else if token.Expiry.Before(time.Now()) {
		log.Print("token is expired. logging in..")
		token = login()
	} else {
		log.Println("Using token from file")
	}

	return token
}

func login() *oauth2.Token {
	http.HandleFunc("/callback", completeLoginCallback)
	go http.ListenAndServe(":8080", nil)

	url := auth.AuthURL(state)
	openURLInBrowser(url)

	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode > 400 {
		log.Fatalf("auth url return code %d", res.StatusCode)
	}

	// wait for auth to complete
	token := <-ch
	return token
}

func openURLInBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		log.Fatal("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func completeLoginCallback(w http.ResponseWriter, r *http.Request) {
	token, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}

	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	storeOAuthTokenInFile(token)

	fmt.Fprintf(w, "Login completed. You can close this tab now.")
	ch <- token
}

func storeOAuthTokenInFile(token *oauth2.Token) {
	file, err := os.Create(credentialFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(token)
}

func retrieveOAuthTokenFromFile() *oauth2.Token {
	file, err := os.Open(credentialFilePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var token oauth2.Token
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&token)
	if err != nil {
		log.Fatal(err)
	}

	return &token
}
