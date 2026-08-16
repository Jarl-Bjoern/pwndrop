package api

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"

	"github.com/kgretzky/pwndrop/log"
	"github.com/kgretzky/pwndrop/storage"
	"github.com/kgretzky/pwndrop/utils"

	"github.com/pquerna/otp"
    "github.com/pquerna/otp/totp"
)

const AUTH_COOKIE_NAME = "t"
const AUTH_SESSION_TIMEOUT_SECS = 24 * 60 * 60

func AuthOptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(
		"Access-Control-Allow-Methods",
		"GET,POST,PUT,PATCH,DELETE,OPTIONS",
	)
}

/*
 * --------------------------------------------------------------------------
 * AUTH CHECK
 * --------------------------------------------------------------------------
 */

func AuthCheckHandler(w http.ResponseWriter, r *http.Request) {
	type AuthResponse struct {
		Status int `json:"status"`
	}

	users, err := storage.UserList()
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	resp := &AuthResponse{}

	/*
	 * No users means first-run/account creation.
	 */
	if len(users) == 0 {
		resp.Status = 0

		DumpResponse(
			w,
			"ok",
			http.StatusOK,
			0,
			resp,
		)
		return
	}

	/*
	 * A partially authenticated MFA session must NOT pass this check.
	 */
	_, err = AuthSession(r)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	resp.Status = 1

	DumpResponse(
		w,
		"ok",
		http.StatusOK,
		0,
		resp,
	)
}

/*
 * --------------------------------------------------------------------------
 * LOGIN
 * --------------------------------------------------------------------------
 */

func LoginUserHandler(w http.ResponseWriter, r *http.Request) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type LoginResponse struct {
		Username    string `json:"username"`
		Token       string `json:"token,omitempty"`
		ApiKey      string `json:"apikey,omitempty"`
		MFARequired bool   `json:"mfa_required"`
	}

	var j LoginRequest

	err := json.NewDecoder(r.Body).Decode(&j)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusBadRequest,
			API_ERROR_BAD_REQUEST,
			nil,
		)
		return
	}

	log.Debug("username: %s", j.Username)

	user, err := storage.UserGetByName(j.Username)
	if err != nil {
		DumpResponse(
			w,
			"invalid username or password",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	/*
	 * Check password.
	 */
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(j.Password),
	)

	if err != nil {
		DumpResponse(
			w,
			"invalid username or password",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	/*
	 * Generate a fresh session for this login attempt.
	 *
	 * IMPORTANT:
	 *
	 * If MFA is enabled, this session is initially NOT authenticated.
	 * /mfa/verify will later change it to authenticated.
	 */
	token := utils.GenRandomHash()

	session := &storage.DbSession{
		Uid:        user.ID,
		Token:      token,
		CreateTime: time.Now().Unix(),

		/*
		 * Password authentication alone is not enough when MFA
		 * is enabled.
		 */
		Authenticated: !user.MFAEnabled,
	}

	_, err = storage.SessionCreate(session)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	/*
	 * Always send the session cookie.
	 *
	 * For MFA users it represents a temporary/unauthenticated
	 * login session.
	 */
	ck := &http.Cookie{
		Domain:   "",
		Path:     "/",
		MaxAge:   AUTH_SESSION_TIMEOUT_SECS,
		HttpOnly: true,
		Name:     AUTH_COOKIE_NAME,
		Value:    token,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, ck)

	/*
	 * MFA required.
	 */
	if user.MFAEnabled {
		DumpResponse(
			w,
			"mfa required",
			http.StatusUnauthorized,
			API_ERROR_MFA_REQUIRED,
			&LoginResponse{
				Username:    user.Name,
				MFARequired: true,
			},
		)
		return
	}

	/*
	 * Normal login.
	 */
	resp := &LoginResponse{
		Username:    user.Name,
		Token:       token,
		ApiKey:      user.ApiKey,
		MFARequired: false,
	}

	DumpResponse(
		w,
		"ok",
		http.StatusOK,
		0,
		resp,
	)
}

/*
 * --------------------------------------------------------------------------
 * LOGOUT
 * --------------------------------------------------------------------------
 */

func LogoutUserHandler(w http.ResponseWriter, r *http.Request) {
	ck, err := r.Cookie(AUTH_COOKIE_NAME)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	session, err := storage.SessionGetByToken(ck.Value)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	err = storage.SessionDelete(session.ID)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	deleteCookie(AUTH_COOKIE_NAME, w)

	DumpResponse(
		w,
		"ok",
		http.StatusOK,
		0,
		nil,
	)
}

/*
 * --------------------------------------------------------------------------
 * AUTH SESSION
 * --------------------------------------------------------------------------
 */

func AuthSession(r *http.Request) (int, error) {
	ck, err := r.Cookie(AUTH_COOKIE_NAME)
	if err != nil {
		return -1, err
	}

	session, err := storage.SessionGetByToken(ck.Value)
	if err != nil {
		return -1, err
	}

	/*
	 * MFA has not been completed.
	 */
	if !session.Authenticated {
		return -1, fmt.Errorf("MFA authentication required")
	}

	/*
	 * Session expiration.
	 */
	expiration := time.Unix(
		session.CreateTime,
		0,
	).Add(
		AUTH_SESSION_TIMEOUT_SECS * time.Second,
	)

	if time.Now().After(expiration) {
		_ = storage.SessionDelete(session.ID)

		return -1, fmt.Errorf("session token expired")
	}

	return session.Uid, nil
}

/*
 * --------------------------------------------------------------------------
 * API KEY AUTH
 * --------------------------------------------------------------------------
 */

func AuthApiKey(r *http.Request) (int, error) {
	ah := r.Header.Get("Authorization")

	if len(ah) != 32 {
		return -1, fmt.Errorf("Unauthorized")
	}

	user, err := storage.UserGetByApiKey(ah)
	if err != nil {
		return -1, err
	}

	return user.ID, nil
}

/*
 * --------------------------------------------------------------------------
 * DELETE COOKIE
 * --------------------------------------------------------------------------
 */

func deleteCookie(name string, w http.ResponseWriter) {
	ck := &http.Cookie{
		Domain:   "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Name:     name,
		Value:    "",
	}

	http.SetCookie(w, ck)
}


func ClearSecretSessionHandler(w http.ResponseWriter, r *http.Request) {
	cookie_name := Cfg.GetCookieName()
	deleteCookie(cookie_name, w)
	DumpResponse(w, "ok", http.StatusOK, 0, nil)
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	type CreateUserRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	type CreateUserResponse struct {
		Username string `json:"username"`
	}

	users, err := storage.UserList()
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusInternalServerError, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}

	_, err = AuthSession(r)
	if len(users) > 0 && err != nil {
		DumpResponse(w, err.Error(), http.StatusUnauthorized, API_ERROR_BAD_AUTHENTICATION, nil)
		return
	}

	j := CreateUserRequest{}
	err = json.NewDecoder(r.Body).Decode(&j)
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusBadRequest, API_ERROR_BAD_REQUEST, nil)
		return
	}

	if j.Username == "" || j.Password == "" {
		DumpResponse(w, "bad request", http.StatusBadRequest, API_ERROR_BAD_REQUEST, nil)
		return
	}

	_, err = storage.UserGetByName(j.Username)
	if err == nil {
		DumpResponse(w, "user already exists", http.StatusOK, API_ERROR_USER_ALREADY_EXISTS, nil)
		return
	}

	phash, err := bcrypt.GenerateFromPassword([]byte(j.Password), 10)
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusBadRequest, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}

	uak := utils.GenRandomString(32)
	o := &storage.DbUser{
		Name:     j.Username,
		Password: string(phash),
		ApiKey: uak,
	}

	_, err = storage.UserCreate(o)
	if err != nil {
		DumpResponse(w, err.Error(), http.StatusBadRequest, API_ERROR_FILE_DATABASE_FAILED, nil)
		return
	}

	resp := &CreateUserResponse{
		Username: j.Username,
	}
	DumpResponse(w, "ok", http.StatusOK, 0, resp)
}


/*
 * --------------------------------------------------------------------------
 * MFA SETUP
 * --------------------------------------------------------------------------
 *
 * This endpoint is called by an already authenticated user.
 *
 * It generates a NEW pending secret.
 *
 * The pending secret is not MFASecret yet.
 * It only becomes MFASecret after the user confirms a valid code.
 */

func MFASetupHandler(w http.ResponseWriter, r *http.Request) {
	uid, err := AuthSession(r)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	user, err := storage.UserGet(uid)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	/*
	 * Don't generate another setup secret if MFA is already active.
	 */
	if user.MFAEnabled {
		DumpResponse(
			w,
			"MFA already enabled",
			http.StatusBadRequest,
			API_ERROR_BAD_REQUEST,
			nil,
		)
		return
	}

	/*
	 * Generate the TOTP secret.
	 */
	key, err := totp.Generate(
		totp.GenerateOpts{
			Issuer:      "PwnDropDB",
			AccountName: user.Name,

			/*
			 * Standard TOTP period.
			 */
			Period: 30,

			SecretSize: 20,
			Digits:     otp.DigitsSix,
			Algorithm:  otp.AlgorithmSHA1,
		},
	)

	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	/*
	 * Store only as pending.
	 *
	 * This means pressing "Enable MFA" does NOT immediately activate
	 * MFA. The user must prove they can generate a valid TOTP code.
	 */
	user.MFAPending = key.Secret()

	_, err = storage.UserUpdate(user)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	DumpResponse(
		w,
		"ok",
		http.StatusOK,
		0,
		map[string]string{
			"secret": key.Secret(),

			/*
			 * This is the complete otpauth:// URL.
			 * The frontend should put THIS into the QR code.
			 */
			"url": key.URL(),
		},
	)
}

/*
 * --------------------------------------------------------------------------
 * MFA ENABLE
 * --------------------------------------------------------------------------
 *
 * Called after the user scans the QR code and enters the generated code.
 */

func MFAEnableHandler(w http.ResponseWriter, r *http.Request) {
	uid, err := AuthSession(r)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	user, err := storage.UserGet(uid)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	if user.MFAEnabled {
		DumpResponse(
			w,
			"MFA is already enabled",
			http.StatusBadRequest,
			API_ERROR_BAD_REQUEST,
			nil,
		)
		return
	}

	var request struct {
		Code string `json:"code"`
	}

	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		DumpResponse(
			w,
			"bad request",
			http.StatusBadRequest,
			API_ERROR_BAD_REQUEST,
			nil,
		)
		return
	}

	/*
	 * There must be a pending secret.
	 */
	if user.MFAPending == "" {
		DumpResponse(
			w,
			"MFA setup has not been started",
			http.StatusBadRequest,
			API_ERROR_BAD_REQUEST,
			nil,
		)
		return
	}

	/*
	 * Validate against MFAPending, NOT MFASecret.
	 */
	valid := totp.Validate(
		request.Code,
		user.MFAPending,
	)

	if !valid {
		DumpResponse(
			w,
			"invalid authenticator code",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	/*
	 * Promote pending secret to active secret.
	 */
	user.MFASecret = user.MFAPending
	user.MFAPending = ""
	user.MFAEnabled = true

	_, err = storage.UserUpdate(user)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	DumpResponse(
		w,
		"MFA enabled",
		http.StatusOK,
		0,
		nil,
	)
}

/*
 * --------------------------------------------------------------------------
 * MFA VERIFY DURING LOGIN
 * --------------------------------------------------------------------------
 */

func MFAVerifyHandler(w http.ResponseWriter, r *http.Request) {
	/*
	 * The login endpoint created this cookie/session, but marked it
	 * Authenticated=false.
	 */
	ck, err := r.Cookie(AUTH_COOKIE_NAME)
	if err != nil {
		DumpResponse(
			w,
			"authentication required",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	session, err := storage.SessionGetByToken(ck.Value)
	if err != nil {
		DumpResponse(
			w,
			"authentication required",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	/*
	 * Check session expiration even though MFA isn't completed yet.
	 */
	expiration := time.Unix(
		session.CreateTime,
		0,
	).Add(
		AUTH_SESSION_TIMEOUT_SECS * time.Second,
	)

	if time.Now().After(expiration) {
		_ = storage.SessionDelete(session.ID)

		DumpResponse(
			w,
			"session token expired",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	user, err := storage.UserGet(session.Uid)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	if !user.MFAEnabled || user.MFASecret == "" {
		DumpResponse(
			w,
			"MFA is not enabled",
			http.StatusBadRequest,
			API_ERROR_BAD_REQUEST,
			nil,
		)
		return
	}

	var request struct {
		Code string `json:"code"`
	}

	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		DumpResponse(
			w,
			"bad request",
			http.StatusBadRequest,
			API_ERROR_BAD_REQUEST,
			nil,
		)
		return
	}

	/*
	 * Basic input validation.
	 */
	if len(request.Code) != 6 {
		DumpResponse(
			w,
			"invalid MFA code",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	/*
	 * IMPORTANT:
	 *
	 * MFA login uses MFASecret.
	 *
	 * MFA setup uses MFAPending.
	 */
	valid := totp.Validate(
		request.Code,
		user.MFASecret,
	)

	if !valid {
		DumpResponse(
			w,
			"invalid MFA code",
			http.StatusUnauthorized,
			API_ERROR_BAD_AUTHENTICATION,
			nil,
		)
		return
	}

	/*
	 * MFA is now complete.
	 */
	session.Authenticated = true

	/*
	 * This assumes your existing SessionCreate implementation updates/
	 * replaces an existing session. If SessionCreate is INSERT-only in
	 * your storage implementation, use its corresponding update method
	 * here instead.
	 */
	_, err = storage.SessionCreate(session)
	if err != nil {
		DumpResponse(
			w,
			err.Error(),
			http.StatusInternalServerError,
			API_ERROR_FILE_DATABASE_FAILED,
			nil,
		)
		return
	}

	DumpResponse(
		w,
		"ok",
		http.StatusOK,
		0,
		nil,
	)
}

func MFAStatusHandler(w http.ResponseWriter, r *http.Request) {
    uid, err := AuthSession(r)
    if err != nil {
        DumpResponse(
            w,
            err.Error(),
            http.StatusUnauthorized,
            API_ERROR_BAD_AUTHENTICATION,
            nil,
        )
        return
    }

    user, err := storage.UserGet(uid)
    if err != nil {
        DumpResponse(
            w,
            err.Error(),
            http.StatusInternalServerError,
            API_ERROR_FILE_DATABASE_FAILED,
            nil,
        )
        return
    }

    DumpResponse(
        w,
        "ok",
        http.StatusOK,
        0,
        map[string]bool{
            "enabled": user.MFAEnabled,
        },
    )
}
