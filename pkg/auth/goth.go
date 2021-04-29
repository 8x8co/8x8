package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserinfoEndpoint = "https://www.googleapis.com/oauth2/v3/userinfo"
const callback = "https://8x8.co.tz/auth/google/callback"
const sessionName = "8x8"

type Google struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
}

func (g Google) Config(redirect string) oauth2.Config {
	scopes := make([]string, len(g.Scopes))
	copy(scopes, g.Scopes)
	return oauth2.Config{
		ClientID:     g.ClientID,
		ClientSecret: g.ClientID,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirect,
		Scopes:       scopes,
	}
}

var defaultGoogleConfig = Google{
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	Scopes: []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
}

type GoogleUser struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Picture       string `json:"picture"`
}

func a(ctx context.Context) oauth2.Config {
	return defaultGoogleConfig.Config(callback)
}

var store = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))

func Login(w http.ResponseWriter, r *http.Request) {
	o := a(r.Context())
	session, _ := store.Get(r, sessionName)
	state := base64.URLEncoding.EncodeToString(securecookie.GenerateRandomKey(16))
	session.Values["state"] = state
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u := o.AuthCodeURL(state)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func Callback(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	var state string
	if s := session.Values["state"]; s != nil {
		state = s.(string)
	}
	if r.FormValue("state") != state {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	o := a(r.Context())
	token, err := o.Exchange(r.Context(), r.FormValue("code"))
	if err != nil {
		return
	}
	res, err := o.Client(r.Context(), token).Get(googleUserinfoEndpoint)
	if err != nil {
		return
	}
	defer res.Body.Close()
	var usr GoogleUser
	json.NewDecoder(res.Body).Decode(&usr)
}
