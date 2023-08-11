// / Package ...
package main

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	portBackendSelector uint16 = iota + 8090
	portBackendFunctional
)

func main() {
	baseDomain, ok := os.LookupEnv("IFRAIXEDES_BASE_DOMAIN")
	if !ok || baseDomain == "" {
		fmt.Println(
			"you must define IFRAIXEDES_BASE_DOMAIN env var to a not empty value (e.g. IFRAIXEDES_BASE_DOMAIN=selector.example.test)",
		)
		os.Exit(1)
	}

	go serveSatellite("127.0.0.1", portBackendFunctional, "functional", baseDomain)

	serveSatelliteSelector(portBackendSelector, baseDomain)
}

func serveSatelliteSelector(port uint16, baseDomain string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/select", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("serving backend selector - /select")

		backend := r.URL.Query().Get("backend")
		switch backend {
		case "functional":
			w.Header().Set("Location", fmt.Sprintf("http://%s.%s", backend, baseDomain))
		default:
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(fmt.Sprintf("unrecognized backend: %q", backend)))
			return
		}

		cookie := http.Cookie{
			Name:  "selected-backend",
			Value: backend,
		}

		http.SetCookie(w, &cookie)
		w.WriteHeader(http.StatusSeeOther)
	})

	mux.HandleFunc("/unselect", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Println("serving backend selector - /unselect")

		cookie := http.Cookie{
			Name:    "selected-backend",
			Expires: time.Now().Add(-24 * time.Hour),
		}

		http.SetCookie(w, &cookie)
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("serving backend selector - /")

		cookie, err := r.Cookie("selected-backend")
		if err != nil {
			if !errors.Is(err, http.ErrNoCookie) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<p> List of services:<br/><br/><a href='/select?backend=functional'>- Functional</a></p><br/>"))
			writeCookies(w, r)
			return
		}

		switch backend := cookie.Value; backend {
		case "functional":
			w.Header().Set("Location", fmt.Sprintf("http://%s.%s", backend, baseDomain))
			w.WriteHeader(http.StatusSeeOther)
		default:
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(fmt.Sprintf(
				"'satellite' cookie contains an unrecognizable value: %q", cookie.Value,
			)))
		}
	})

	err := http.ListenAndServe(":"+strconv.Itoa(int(port)), mux)
	if err != nil {
		fmt.Printf("Server listener error: %+v\n", err)
	}
}

func serveSatellite(ip string, port uint16, backendID string, baseDomain string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("serving backend: %s - /\n", backendID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("<h1>Backend: %s</h1>", backendID)))
		w.Write([]byte("<a href='/sub-page'>Got to sub-page</a><br/>"))
		w.Write([]byte("<a href='https://" + baseDomain + "/unselect'>Go back to select another backend</a>"))
		writeCookies(w, r)
	})

	mux.HandleFunc("/sub-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("serving backend: %s - /sub-page\n", backendID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("<h1>Backend: %s</h1>", backendID)))
		w.Write([]byte("<a href='https://" + baseDomain + "/unselect'>Go back to select another backend</a>"))
		writeCookies(w, r)
	})

	err := http.ListenAndServe(ip+":"+strconv.Itoa(int(port)), mux)
	if err != nil {
		fmt.Printf("Server listener error: %+v\n", err)
	}
}

func writeCookies(w http.ResponseWriter, r *http.Request) {
	data := "<pre>\nList of cookies:\n\n"
	for h, v := range r.Header {
		data += fmt.Sprintf("%s: %s\n", h, v)
	}
	data += "</pre>"

	w.Write([]byte(data))
}
