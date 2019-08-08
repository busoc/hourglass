package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/busoc/hourglass"
	"github.com/gorilla/handlers"
	"github.com/midbel/jwt"
)

type Func func(*http.Request) (interface{}, error)

func handle(f Func, w io.Writer, s jwt.Signer) http.Handler {
	// var h http.Handler
	// if s != nil {
	// } else {
	// 	h = negociate(f)
	// }
	h := cors(authorize(negociate(f), s))
	return handlers.LoggingHandler(w, handlers.CompressHandler(h))
}

func allow(f Func, w io.Writer, u, p string, hosts []string) http.Handler {
	if u == "" && p == "" && len(hosts) == 0 {
		return negociate(f) //handle(f, w, nil)
	}
	sort.Strings(hosts)

	e := negociate(f)
	h := func(w http.ResponseWriter, r *http.Request) {
		if ru, rp, ok := r.BasicAuth(); !ok || (ru != u || rp != p) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if len(hosts) > 0 {
			o, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			ix := sort.SearchStrings(hosts, o)
			if ix >= len(hosts) || hosts[ix] != o {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		e.ServeHTTP(w, r)
	}
	x := http.HandlerFunc(h)
	return handlers.LoggingHandler(w, handlers.CompressHandler(x))
}

func cors(h http.Handler) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if h := r.Header.Get("Access-Control-Request-Headers"); len(h) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", h)
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", r.Header.Get("Access-Control-Request-Method"))
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(handler)
}

func authorize(h http.Handler, s jwt.Signer) http.Handler {
	const (
		bearer = "Bearer "
		basic  = "Basic "
		auth   = "Authorization"
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			h.ServeHTTP(w, r)
			return
		}
		var (
			user *hourglass.User
			err  error
		)
		switch h := r.Header.Get(auth); {
		case strings.HasPrefix(h, basic):
			u, p, _ := r.BasicAuth()
			user, err = hourglass.Authenticate(db, u, p)
		case strings.HasPrefix(h, bearer):
			user = new(hourglass.User)
			err = s.Verify(h[len(bearer):], user)

			if err == nil {
				t, _ := s.Sign(user)
				w.Header().Set(auth, bearer+t)
			}
		default:
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			if !user.Internal {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		default:
		}
		ctx := r.Context()
		h.ServeHTTP(w, r.WithContext(context.WithValue(ctx, "user", user.Initial)))
	}
	return http.HandlerFunc(handler)
}

func negociate(f Func) http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		d, err := f(r)
		switch err {
		case nil:
		case hourglass.ErrNotFound:
			w.WriteHeader(http.StatusNotFound)
			return
		case hourglass.ErrUnauthenticated:
			w.WriteHeader(http.StatusUnauthorized)
			return
		default:
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if d == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(d); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, err = io.Copy(w, buf)
		if err != nil {
			log.Println(err)
		}
	}
	return http.HandlerFunc(handler)
}
