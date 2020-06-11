package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Release holds data from the releases response
type Release struct {
	Tag string `json:"tag_name"`
}

// ReleaseEvent holds data from the github release webhook payload
type ReleaseEvent struct {
	Action string `json:"action"`
}

// Diff holds data from the compare response
type Diff struct {
	Commits []struct {
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	} `json:"commits"`
}

// Req is the json payload sent to the webhook
type Req struct {
	Content string `json:"content"`
}

func main() {

	port := os.Getenv("port")
	secret := os.Getenv("secret")
	owner := os.Getenv("github_owner")
	repo := os.Getenv("github_repo")
	webhook := os.Getenv("webhook")

	if port == "" || secret == "" || owner == "" || repo == "" || webhook == "" {
		log.Fatal("Missing environment variables")
	}

	http.HandleFunc("/github/release", func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			if r := recover(); r != nil {
				log.Printf("error: %v", r)
			}
		}()

		if r.Method != "POST" {
			http.Error(w, "Wrong HTTP method", http.StatusBadRequest)
			return
		}

		secrets, ok := r.URL.Query()["secret"]

		if !ok || len(secrets) < 1 {
			http.Error(w, "Missing secret", http.StatusUnauthorized)
			return
		}

		if secrets[0] != secret {
			http.Error(w, "Wrong secret", http.StatusForbidden)
			return
		}

		var releaseEvent ReleaseEvent

		err := json.NewDecoder(r.Body).Decode(&releaseEvent)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if releaseEvent.Action != "published" {
			http.Error(w, "Action must be published", http.StatusBadRequest)
			return
		}

		// Wait 5 seconds to make sure github's api is up to date
		time.Sleep(time.Second * 5)

		// Get releases
		releasesRes, err := http.Get("https://api.github.com/repos/" + owner + "/" + repo + "/releases")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var releases []Release

		err = json.NewDecoder(releasesRes.Body).Decode(&releases)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Compare commits
		diffRes, err := http.Get("https://api.github.com/repos/" + owner + "/" + repo + "/compare/" + releases[1].Tag + "..." + releases[0].Tag)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var diff Diff

		err = json.NewDecoder(diffRes.Body).Decode(&diff)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		message := "**" + releases[0].Tag + " release**\n```"

		for _, c := range diff.Commits {

			commit := strings.Split(c.Commit.Message, "\n")
			if len(commit) > 0 {
				message += "\n• " + commit[0]
			}

		}

		message += "\n```"

		req, err := json.Marshal(Req{Content: message})

		if err != nil {
			return
		}

		resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(req))

		if err != nil {
			log.Fatal(err.Error())
		}

		resp.Body.Close()

		fmt.Fprint(w, "Success")

	})

	log.Println("Listening on port " + port)

	log.Fatal(http.ListenAndServe(":"+port, nil))

}
