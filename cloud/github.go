package cloud

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHub wraps the small slice of the GitHub API the cloud needs: OAuth login and
// reading manifest files from a repository. Uses only the standard library.
type GitHub struct {
	ClientID     string
	ClientSecret string
	http         *http.Client
}

// NewGitHub builds a GitHub client. ClientID/Secret are required only for the
// OAuth login flow; public-repo manifest reads work without them.
func NewGitHub(clientID, clientSecret string) *GitHub {
	return &GitHub{ClientID: clientID, ClientSecret: clientSecret, http: &http.Client{Timeout: 20 * time.Second}}
}

// Configured reports whether OAuth login is available (client id + secret set).
func (g *GitHub) Configured() bool { return g.ClientID != "" && g.ClientSecret != "" }

// AuthURL is the GitHub authorize URL to redirect the user to. `read:user` is the
// only scope needed for login; add `repo` when private-repo sync is enabled.
func (g *GitHub) AuthURL(redirectURI, state, scope string) string {
	if scope == "" {
		scope = "read:user user:email"
	}
	q := url.Values{
		"client_id":    {g.ClientID},
		"redirect_uri": {redirectURI},
		"scope":        {scope},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + q.Encode()
}

// Exchange trades an OAuth code for an access token.
func (g *GitHub) Exchange(ctx context.Context, code, redirectURI string) (string, error) {
	form := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := g.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error_description"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("github oauth: %s", firstNonEmpty(out.Error, "no access token returned"))
	}
	return out.AccessToken, nil
}

// FetchUser loads the authenticated user's profile for the given token.
func (g *GitHub) FetchUser(ctx context.Context, token string) (User, error) {
	var u struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := g.get(ctx, token, "https://api.github.com/user", &u); err != nil {
		return User{}, err
	}
	return User{
		ID: userID("gh", u.ID), Login: u.Login, Name: u.Name,
		AvatarURL: u.AvatarURL, Email: u.Email, CreatedAt: time.Now().UTC(),
	}, nil
}

// FetchManifests reads YAML manifest files from a repository (optionally under a
// subpath) and returns them keyed by repo path, plus the resolved commit SHA. A
// token is required for private repos; public repos work with an empty token.
func (g *GitHub) FetchManifests(ctx context.Context, repo, ref, subpath, token string) (map[string]string, string, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return nil, "", fmt.Errorf("repo must be owner/name, got %q", repo)
	}
	if ref == "" {
		ref = "main"
	}

	// Resolve the ref to a commit SHA (records exactly what was analyzed).
	var commit struct {
		SHA string `json:"sha"`
	}
	_ = g.get(ctx, token, fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, name, url.PathEscape(ref)), &commit)

	// List the tree recursively and select manifest files under subpath.
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	treeRef := ref
	if commit.SHA != "" {
		treeRef = commit.SHA
	}
	if err := g.get(ctx, token, fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, name, url.PathEscape(treeRef)), &tree); err != nil {
		return nil, "", fmt.Errorf("list repo tree: %w", err)
	}

	const maxFiles = 200
	out := map[string]string{}
	for _, e := range tree.Tree {
		if e.Type != "blob" || !isManifestFile(e.Path) {
			continue
		}
		if subpath != "" && !strings.HasPrefix(e.Path, strings.Trim(subpath, "/")+"/") && e.Path != strings.Trim(subpath, "/") {
			continue
		}
		content, err := g.fetchContent(ctx, owner, name, e.Path, treeRef, token)
		if err != nil {
			continue // skip unreadable files rather than failing the whole sync
		}
		out[e.Path] = content
		if len(out) >= maxFiles {
			break
		}
	}
	if len(out) == 0 {
		return nil, commit.SHA, fmt.Errorf("no YAML manifests found in %s@%s%s", repo, ref, pathSuffix(subpath))
	}
	return out, commit.SHA, nil
}

func (g *GitHub) fetchContent(ctx context.Context, owner, name, path, ref, token string) (string, error) {
	var c struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, name, path, url.QueryEscape(ref))
	if err := g.get(ctx, token, u, &c); err != nil {
		return "", err
	}
	if c.Encoding == "base64" {
		b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(c.Content, "\n", ""))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return c.Content, nil
}

func (g *GitHub) get(ctx context.Context, token, u string, v any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("github api %s: %s", u, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(v)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func pathSuffix(p string) string {
	if p == "" {
		return ""
	}
	return " (" + p + ")"
}
