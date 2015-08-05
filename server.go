package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zamN/spotifun/_third_party/github.com/zamN/zamn.net/_third_party/github.com/go-martini/martini"
	"github.com/zamN/spotifun/_third_party/github.com/zamN/zamn.net/_third_party/github.com/martini-contrib/render"
)

type SpotifyMeta struct {
	RedirectUri  string `json:"redirect_uri"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type SpotifyAuth struct {
	AccessToken  string  `json:"access_token"`
	TokenType    string  `json:"token_type"`
	ExpiresIn    float64 `json:"expires_in"`
	RefreshToken string  `json:"refresh_token"`
}

type SpotifyError struct {
	ErrorName        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (se SpotifyError) Error() string {
	return se.ErrorName + ": " + se.ErrorDescription
}

var sAuth SpotifyAuth
var sError SpotifyError
var sMeta SpotifyMeta

func (sa *SpotifyAuth) spotifyAccess(code string, grant_type string) error {
	client := &http.Client{}

	file, err := ioutil.ReadFile("./spotifymeta.json")
	if err != nil {
		fmt.Printf("Error %s", err)
		return err
	}

	err = json.Unmarshal(file, &sMeta)
	if err != nil {
		fmt.Println("error:", err)
	}

	vals := url.Values{}
	vals.Add("grant_type", grant_type)
	vals.Add("code", code)
	vals.Add("redirect_uri", sMeta.RedirectUri)

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(vals.Encode()))

	if err != nil {
		fmt.Printf("Error contacting Spotify API: %s\n", err)
	}

	// Specification calls for Authorization format: Basic client_id:client_secret
	b64idSec := base64.StdEncoding.EncodeToString([]byte(sMeta.ClientId + ":" + sMeta.ClientSecret))

	req.Header.Add("Authorization", "Basic "+b64idSec)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		err = json.Unmarshal(body, sa)
		if err != nil {
			fmt.Println("error:", err)
		}

		// Casting from float64 to int64 feels so..wrong
		time.AfterFunc(time.Duration(sa.ExpiresIn)*time.Second, sa.refreshToken)

		return nil
	} else {
		var sError SpotifyError

		err = json.Unmarshal(body, &sError)
		if err != nil {
			fmt.Println("error:", err)
		}

		return sError
	}
}

func spotifyLogic(r *http.Request) string {
	q := r.URL.Query()
	code := q.Get("code")
	spot_error := q.Get("error")
	state := q.Get("state")

	if state != "" {
		fmt.Println(state)
	}

	if spot_error != "" {
		return spot_error
	}

	// User authorized our application, now lets get API access!
	if code != "" {
		err := sAuth.spotifyAccess(code, "authorization_code")
		if err != nil {
			return err.Error()
		}
		fmt.Println(sAuth)

		/*
		   req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(vals.Encode()))

		   if err != nil {
		       fmt.Printf("Error contacting Spotify API: %s\n", err)
		   }

		   req.Header.Add("Authorization", "Basic MzZhOGQ0MGEyZjUyNDk5MjliZTc5NDE0ZTdjMjVkMWQ6Y2RlNjFjYzczYjQyNGNlMjlhNmY1ZjQ2MTQ2YTkzYjk=")
		   req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		*/

		return code
	}

	return "hi!"
}

func (sa *SpotifyAuth) refreshToken() {
	sa.spotifyAccess(sa.RefreshToken, "refresh_token")
}

func main() {
	// Set the port via the PORT env var
	fmt.Println("Running server..")
	m := martini.Classic()
	m.Use(render.Renderer(render.Options{
		Directory:  "templates",
		Layout:     "main",
		Extensions: []string{".tmpl", ".html"},
		Charset:    "UTF-8",
		IndentJSON: true,
	}))

	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", nil)
	})

	m.Get("/spotify", spotifyLogic)
	m.Post("/spotify", spotifyLogic)
	m.Run()
}
