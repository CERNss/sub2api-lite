package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CERNss/sub2api-lite/ent/authidentity"
	dbuser "github.com/CERNss/sub2api-lite/ent/user"
	"github.com/CERNss/sub2api-lite/internal/config"
	"github.com/CERNss/sub2api-lite/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEmailOAuthCallbackRequiresPendingRegistrationWhenInvitationEnabled(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandler(t, true)
	ctx := context.Background()

	state := "github-oauth-state"
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/github/callback?code=code-1&state="+url.QueryEscape(state), nil)
	req.AddCookie(&http.Cookie{Name: emailOAuthStateCookieName, Value: encodeCookieValue(state)})
	req.AddCookie(&http.Cookie{Name: emailOAuthRedirectCookie, Value: encodeCookieValue("/dashboard")})
	req.AddCookie(&http.Cookie{Name: emailOAuthProviderCookie, Value: encodeCookieValue("github")})
	c.Request = req

	profile := &emailOAuthProfile{
		Subject:       "github-123",
		Email:         "fresh@example.com",
		EmailVerified: true,
		Username:      "fresh",
		DisplayName:   "Fresh User",
		AvatarURL:     "https://cdn.example/fresh.png",
		Metadata: map[string]any{
			"login": "fresh",
		},
	}
	handler.emailOAuthCallbackWithProfile(c, "github", config.EmailOAuthProviderConfig{
		Enabled:             true,
		ClientID:            "github-client",
		ClientSecret:        "github-secret",
		RedirectURL:         "https://app.example/api/v1/auth/oauth/github/callback",
		FrontendRedirectURL: "/auth/oauth/callback",
	}, "/auth/oauth/callback", "/dashboard", profile)

	require.Equal(t, http.StatusFound, recorder.Code)
	location := recorder.Header().Get("Location")
	require.Contains(t, location, "/auth/oauth/callback")
	require.NotContains(t, location, "access_token=")

	userCount, err := client.User.Query().Where(dbuser.EmailEQ("fresh@example.com")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, userCount)

	session, err := client.PendingAuthSession.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "github", session.ProviderType)
	require.Equal(t, "github", session.ProviderKey)
	require.Equal(t, "github-123", session.ProviderSubject)
	require.Equal(t, "fresh@example.com", session.ResolvedEmail)
	require.Equal(t, "/dashboard", session.RedirectTo)
	require.Nil(t, session.TargetUserID)

	completion, ok := readCompletionResponse(session.LocalFlowState)
	require.True(t, ok)
	require.Equal(t, oauthPendingChoiceStep, completion["step"])
	require.Equal(t, "invitation_required", completion["error"])
	require.Equal(t, true, completion["invitation_required"])
	require.Equal(t, "fresh@example.com", completion["email"])
	require.Equal(t, "fresh@example.com", completion["resolved_email"])
	require.Equal(t, true, completion["create_account_allowed"])

	require.NotEmpty(t, findSetCookieValue(recorder.Result().Cookies(), oauthPendingSessionCookieName))
	require.NotEmpty(t, findSetCookieValue(recorder.Result().Cookies(), oauthPendingBrowserCookieName))
}

func TestEmailOAuthCallbackExistingEmailLogsInWhenInvitationEnabled(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandler(t, true)
	ctx := context.Background()

	user, err := client.User.Create().
		SetEmail("existing@example.com").
		SetUsername("existing").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/callback", nil)

	handler.emailOAuthCallbackWithProfile(c, "google", config.EmailOAuthProviderConfig{
		Enabled:             true,
		ClientID:            "google-client",
		ClientSecret:        "google-secret",
		RedirectURL:         "https://app.example/api/v1/auth/oauth/google/callback",
		FrontendRedirectURL: "/auth/oauth/callback",
	}, "/auth/oauth/callback", "/dashboard", &emailOAuthProfile{
		Subject:       "google-123",
		Email:         "existing@example.com",
		EmailVerified: true,
		Username:      "existing",
	})

	require.Equal(t, http.StatusFound, recorder.Code)
	location := recorder.Header().Get("Location")
	require.Contains(t, location, "access_token=")
	require.Contains(t, location, "redirect=%252Fdashboard")

	sessionCount, err := client.PendingAuthSession.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, sessionCount)

	identityCount, err := client.AuthIdentity.Query().Where(
		authidentity.ProviderTypeEQ("google"),
		authidentity.ProviderSubjectEQ("google-123"),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, identityCount)
	_ = user
}

func TestCompleteEmailOAuthRegistrationRequiresPassword(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandler(t, false)
	ctx := context.Background()

	session, err := client.PendingAuthSession.Create().
		SetSessionToken("email-oauth-password-session-token").
		SetIntent(oauthIntentLogin).
		SetProviderType("github").
		SetProviderKey("github").
		SetProviderSubject("github-password-user").
		SetResolvedEmail("password-required@example.com").
		SetRedirectTo("/dashboard").
		SetBrowserSessionKey("browser-password-key").
		SetUpstreamIdentityClaims(map[string]any{
			"email":            "password-required@example.com",
			"email_verified":   true,
			"username":         "password-required",
			"provider":         "github",
			"provider_key":     "github",
			"provider_subject": "github-password-user",
		}).
		SetLocalFlowState(map[string]any{
			"step":  oauthPendingChoiceStep,
			"error": "registration_completion_required",
		}).
		SetExpiresAt(time.Now().UTC().Add(10 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/github/complete-registration", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: oauthPendingSessionCookieName, Value: encodeCookieValue(session.SessionToken)})
	req.AddCookie(&http.Cookie{Name: oauthPendingBrowserCookieName, Value: encodeCookieValue("browser-password-key")})
	c.Request = req

	handler.completeEmailOAuthRegistration(c, "github")

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	userCount, err := client.User.Query().Where(dbuser.EmailEQ("password-required@example.com")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, userCount)
}

func TestParseGitHubOAuthProfileRejectsPublicEmailWhenEmailsEndpointFails(t *testing.T) {
	emailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing scope", http.StatusForbidden)
	}))
	t.Cleanup(emailServer.Close)

	profile, err := parseGitHubOAuthProfile(context.Background(), config.EmailOAuthProviderConfig{
		EmailsURL: emailServer.URL,
	}, &emailOAuthTokenResponse{AccessToken: "token"}, `{"id":123,"login":"octo","email":"public@example.com"}`)

	require.Error(t, err)
	require.Nil(t, profile)
	require.Contains(t, err.Error(), "github emails endpoint status 403")
}

type oauthEmailAffiliateBindCall struct {
	userID    int64
	inviterID int64
}

func findSetCookieValue(cookies []*http.Cookie, name string) string {
	for _, cookie := range cookies {
		if cookie != nil && strings.EqualFold(cookie.Name, name) && cookie.MaxAge >= 0 {
			return cookie.Value
		}
	}
	return ""
}
