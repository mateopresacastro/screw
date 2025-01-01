package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"tagg/cryptoutil"
	"tagg/session"
	"tagg/store"
	"time"
)

const (
	googleAuthorizeUrl         = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenUrl             = "https://oauth2.googleapis.com/token"
	googleUserInfoURL          = "https://www.googleapis.com/oauth2/v2/userinfo"
	googleOAuthStateCookieName = "google_oauth_state"
)

var scopes = []string{
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/userinfo.email",
}

type google struct {
	clientID     string
	clientSecret string
	callbackURL  string
	store        store.Store
	sessionMgr   *session.Manager
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

func NewGoogle(clientID, clientSecret, callbackURL string, store store.Store, sessionMgr *session.Manager) *google {
	return &google{
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
		store:        store,
		sessionMgr:   sessionMgr,
	}
}

func (g *google) HandleLogin(w http.ResponseWriter, r *http.Request) {
	authorizationURL, err := url.Parse(googleAuthorizeUrl)
	if err != nil {
		slog.Error("Failed to parse Google authorization URL", "error", err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	state, err := createState()
	if err != nil {
		slog.Error("Failed to create OAuth state", "error", err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}

	query := authorizationURL.Query()
	query.Set("state", state)
	query.Set("client_id", g.clientID)
	query.Set("redirect_uri", g.callbackURL)
	query.Set("response_type", "code")

	query.Set("scope", strings.Join(scopes, " "))

	authorizationURL.RawQuery = query.Encode()

	http.SetCookie(w, &http.Cookie{
		Name:     googleOAuthStateCookieName,
		Value:    state,
		MaxAge:   int(10 * time.Minute),
		HttpOnly: true,
		Secure:   false, // change when https
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, authorizationURL.String(), http.StatusFound)
}

func (g *google) HandleCallBack(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")
	storedState, err := r.Cookie(googleOAuthStateCookieName)
	if err != nil || storedState.Value != state || code == "" {
		slog.Error("Invalid OAuth state or missing code",
			"error", err,
			"state_match", storedState != nil && storedState.Value == state,
			"has_code", code != "")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     googleOAuthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	formData := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {g.callbackURL},
	}

	req, err := http.NewRequest("POST", googleTokenUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		slog.Error("Failed to create token request", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	basicAuth := base64.StdEncoding.EncodeToString([]byte(g.clientID + ":" + g.clientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth)
	client := &http.Client{Timeout: 10 * time.Second}

	tokenResp, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to execute token request", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		slog.Error("Token endpoint returned non-200 status", "status", tokenResp.StatusCode)
		http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
		return
	}

	var tokenRespData tokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenRespData); err != nil {
		slog.Error("Failed to decode token response", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	userReq, err := http.NewRequest("GET", googleUserInfoURL, nil)
	if err != nil {
		slog.Error("Failed to create user info request", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	userReq.Header.Set("Authorization", "Bearer "+tokenRespData.AccessToken)
	userResp, err := client.Do(userReq)
	if err != nil {
		slog.Error("Failed to execute user info request", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()

	var userData struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&userData); err != nil {
		slog.Error("Failed to decode user info response", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if userData.VerifiedEmail == false {
		slog.Error("User email not verified", "email", userData.Email)
		http.Error(w, "Email not verified", http.StatusBadRequest)
		return
	}

	existingUser, err := g.store.UserByGoogleID(userData.ID)
	if err == nil && existingUser != nil {
		newSessionToken, err := cryptoutil.Random()
		if err != nil {
			slog.Error("Failed to generate session token for existing user", "error", err, "user_id", existingUser.ID)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		session, err := g.sessionMgr.CreateSession(newSessionToken, existingUser.ID)
		if err != nil {
			slog.Error("Failed to create session for existing user", "error", err, "user_id", existingUser.ID)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		g.sessionMgr.SetSessionCookie(w, newSessionToken, session.ExpiresAt)
		http.Redirect(w, r, "http://localhost:3001", http.StatusFound)
		return
	}

	if errors.Is(err, store.ErrUserNotFound) == false {
		slog.Error("error reading user from db", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user := &store.User{
		Email:    userData.Email,
		Picture:  userData.Picture,
		Name:     userData.Name,
		GoogleID: userData.ID,
	}

	newUserID, err := g.store.CreateUser(user)
	if err != nil {
		slog.Error("Failed to create new user", "error", err, "email", userData.Email)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	newSessionToken, err := cryptoutil.Random()
	if err != nil {
		slog.Error("Failed to generate session token for new user", "error", err, "user_id", newUserID)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	session, err := g.sessionMgr.CreateSession(newSessionToken, newUserID)
	if err != nil {
		slog.Error("Failed to create session for new user", "error", err, "user_id", newUserID)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	g.sessionMgr.SetSessionCookie(w, newSessionToken, session.ExpiresAt)
	http.Redirect(w, r, "http://localhost:3001", http.StatusPermanentRedirect)
	return
}

func createState() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
