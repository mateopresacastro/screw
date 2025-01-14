package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"tagg/cryptoutil"
	"tagg/herr"
	"tagg/session"
	"tagg/store"
	"time"
)

type google struct {
	clientID               string
	clientSecret           string
	callbackURL            string
	store                  store.Store
	sessionMgr             *session.Manager
	authUrl                string
	tokenUrl               string
	userInfoUrl            string
	stateCookieName        string
	codeVerifierCookieName string
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

func NewGoogle(clientID, clientSecret, callbackURL string, store store.Store, sessionMgr *session.Manager) *google {
	return &google{
		clientID:               clientID,
		clientSecret:           clientSecret,
		callbackURL:            callbackURL,
		store:                  store,
		sessionMgr:             sessionMgr,
		authUrl:                "https://accounts.google.com/o/oauth2/v2/auth",
		tokenUrl:               "https://oauth2.googleapis.com/token",
		userInfoUrl:            "https://www.googleapis.com/oauth2/v2/userinfo",
		stateCookieName:        "google_oauth_state",
		codeVerifierCookieName: "google_code_verifier",
	}
}

func (g *google) HandleLogin(w http.ResponseWriter, r *http.Request) *herr.Error {
	authorizationURL, err := url.Parse(g.authUrl)
	if err != nil {
		return herr.Internal(err, "Failed to parse Google authorization URL")
	}

	state, err := cryptoutil.CreateState()
	if err != nil {
		return herr.Internal(err, "Failed to create OAuth state")
	}

	codeVerifier, err := cryptoutil.CreateCodeVerifier()
	if err != nil {
		return herr.Internal(err, "Failed to create code verifier")
	}

	codeChallenge := cryptoutil.CreateS256CodeChallenge(codeVerifier)

	query := authorizationURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", g.clientID)
	query.Set("redirect_uri", g.callbackURL)
	query.Set("state", state)
	query.Set("code_challenge_method", "S256")
	query.Set("code_challenge", codeChallenge)
	query.Set("scope", "openid profile email")

	authorizationURL.RawQuery = query.Encode()

	http.SetCookie(w, &http.Cookie{
		Name:     g.stateCookieName,
		Value:    state,
		MaxAge:   int(10 * time.Minute),
		HttpOnly: true,
		Secure:   false, // TODO: change when https
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     g.codeVerifierCookieName,
		Value:    codeVerifier,
		MaxAge:   int(10 * time.Minute),
		HttpOnly: true,
		Secure:   false, // TODO: change when https
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, authorizationURL.String(), http.StatusFound)
	return nil
}

func (g *google) HandleCallBack(w http.ResponseWriter, r *http.Request) *herr.Error {
	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")
	stateInCookie, err := r.Cookie(g.stateCookieName)
	if err != nil {
		return herr.BadRequest(err, "Error getting state cookie")
	}

	codeVerifierInCookie, err := r.Cookie(g.codeVerifierCookieName)
	if err != nil {
		return herr.BadRequest(err, "Error getting code verifier cookie")
	}

	if code == "" || state == "" || stateInCookie.Value == "" || codeVerifierInCookie.Value == "" {
		return herr.BadRequest(err, "Missing data")
	}

	if state != stateInCookie.Value {
		return herr.BadRequest(err, "States differ")
	}

	g.deleteCookies(w)

	formData := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {g.callbackURL},
		"code_verifier": {codeVerifierInCookie.Value},
	}

	req, err := http.NewRequest("POST", g.tokenUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		return herr.Internal(err, "Failed to create token request")
	}

	basicAuth := base64.StdEncoding.EncodeToString([]byte(g.clientID + ":" + g.clientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth)
	client := &http.Client{Timeout: 10 * time.Second}

	tokenResp, err := client.Do(req)
	if err != nil {
		return herr.Internal(err, "Failed to execute token request")
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return herr.Internal(errors.New("non-200 status code"), "Token endpoint returned non-200 status")
	}

	var tokenRespData tokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenRespData); err != nil {
		return herr.Internal(err, "Failed to decode token response")
	}

	userReq, err := http.NewRequest("GET", g.userInfoUrl, nil)
	if err != nil {
		return herr.Internal(err, "Failed to create user info request")
	}

	userReq.Header.Set("Authorization", "Bearer "+tokenRespData.AccessToken)
	userResp, err := client.Do(userReq)
	if err != nil {
		return herr.Internal(err, "Failed to execute user info request")
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
		return herr.Internal(err, "Failed to decode user info response")
	}

	if !userData.VerifiedEmail {
		return herr.BadRequest(errors.New("email not verified"), "User email not verified")
	}

	existingUser, err := g.store.UserByGoogleID(userData.ID)
	if err == nil && existingUser != nil {
		err := g.sessionMgr.CreateSession(w, existingUser.ID)
		if err != nil {
			return herr.Internal(err, "Failed to create session for existing user")
		}

		http.Redirect(w, r, "http://localhost", http.StatusFound)
		return nil
	}

	if !errors.Is(err, store.ErrUserNotFound) {
		return herr.Internal(err, "Error reading user from db")
	}

	user := &store.User{
		Email:    userData.Email,
		Picture:  userData.Picture,
		Name:     userData.Name,
		GoogleID: userData.ID,
	}

	newUserID, err := g.store.CreateUser(user)
	if err != nil {
		return herr.Internal(err, "Failed to create new user")
	}

	err = g.sessionMgr.CreateSession(w, newUserID)
	if err != nil {
		return herr.Internal(err, "Failed to create session for new user")
	}

	http.Redirect(w, r, "http://localhost", http.StatusPermanentRedirect)
	return nil
}

func (g *google) deleteCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     g.stateCookieName,
		Value:    "",
		Path:     "/login",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     g.codeVerifierCookieName,
		Value:    "",
		Path:     "/login",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
