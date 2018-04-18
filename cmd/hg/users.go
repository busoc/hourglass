package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/busoc/hourglass"
	"github.com/gorilla/mux"
	"github.com/midbel/jwt"
)

func signin(s jwt.Signer) http.Handler {
	f := func(r *http.Request) (interface{}, error) {
		c := struct {
			User   string `json:"user"`
			Passwd string `json:"passwd"`
		}{}
		if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(&c); err != nil {
			return nil, err
		}
		u, err := hourglass.Authenticate(db, c.User, c.Passwd)
		if err != nil {
			return nil, err
		}
		t, _ := s.Sign(u)
		a := struct {
			*hourglass.User
			Token string `json:"token"`
		}{u, t}
		return a, nil
	}
	return cors(negociate(f))
}

func listUsers(r *http.Request) (interface{}, error) {
	return hourglass.ListUsers(db)
}

func registerUser(r *http.Request) (interface{}, error) {
	v := struct {
		*hourglass.User
		Passwd string `json:"passwd"`
	}{}
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(&v); err != nil {
		return nil, err
	}
	if err := hourglass.RegisterUser(db, v.User, v.Passwd); err != nil {
		return nil, err
	}
	return v.User, nil
}

func updateUser(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	u, err := hourglass.ViewUser(db, id)
	if err != nil {
		return nil, err
	}
	v := &hourglass.User{
		First:    u.First,
		Last:     u.Last,
		Initial:  u.Initial,
		Internal: u.Internal,
		Email:    u.Email,
		Settings: u.Settings,
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(v); err != nil {
		return nil, err
	}

	v.Id = id
	return hourglass.UpdateUser(db, v)
}

func updatePasswd(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	u, err := hourglass.ViewUser(db, id)
	if err != nil {
		return nil, err
	}
	a := r.Context().Value("user").(string)
	if a != u.Initial {
		return nil, fmt.Errorf("forbidden")
	}
	v := struct {
		Old string `json:"old"`
		New string `json:"new"`
	}{}
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(&v); err != nil {
		return nil, err
	}
	return nil, hourglass.UpdatePasswd(db, u, v.Old, v.New)
}

func viewUser(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	return hourglass.ViewUser(db, id)
}
