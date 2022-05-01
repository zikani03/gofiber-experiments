package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/gologin/v2"
	"github.com/dghubble/gologin/v2/github"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/django"
	gogithub "github.com/google/go-github/v44/github"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"
)

func staticPage(templateFile, title string) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.Render(templateFile, fiber.Map{"Title": title})
	}
}

// issueSession issues a cookie session after successful Github login
func issueSession(store *session.Store) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		githubUser, err := github.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(githubUser)
		if err == nil {
			store.Storage.Set("auth.user", data, time.Duration(24*time.Hour))
		}
		http.Redirect(w, req, "/home", http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

func main() {
	// 1. Register LoginHandler and CallbackHandler
	oauth2Config := &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_KEY"),
		ClientSecret: os.Getenv("GITHUB_SECRET"),
		RedirectURL:  os.Getenv("AUTH_CALLBACK_URL"),
		Endpoint:     githubOAuth2.Endpoint,
	}
	// state param cookies require HTTPS by default; disable for localhost development
	stateConfig := gologin.DebugOnlyCookieConfig

	store := session.New()
	engine := django.New("./templates", ".html")

	engine.Debug(true)

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login", http.StatusPermanentRedirect)
	})

	app.Get("/login", staticPage("login", "Login"))
	app.Get("/auth/error", staticPage("auth_error", "Auth Error"))

	app.Get("/auth/github", adaptor.HTTPHandler(github.StateHandler(stateConfig, github.LoginHandler(oauth2Config, nil))))

	githubCallback := adaptor.HTTPHandler(github.StateHandler(stateConfig, github.CallbackHandler(oauth2Config, issueSession(store), nil)))

	app.Get("/auth/github/callback", githubCallback)

	githubAuthMiddleware := func(c *fiber.Ctx) error {
		userJson, err := store.Storage.Get("auth.user")
		if err != nil {
			return c.Redirect("/auth/error")
		}
		user := gogithub.User{}
		json.Unmarshal(userJson, &user)
		c.Locals("user", user)

		return c.Next()
	}

	app.Get("/home", githubAuthMiddleware, func(c *fiber.Ctx) error {
		user := c.Locals("user").(gogithub.User)
		return c.Render("userinfo", fiber.Map{
			"Title": "User Info",
			"user":  &user,
		})
	})

	app.Get("/auth/logout", func(c *fiber.Ctx) error {
		store.Storage.Delete("auth.user")
		return c.Redirect("/", http.StatusTemporaryRedirect)
	})

	err := app.Listen(":3000")
	if err != nil {
		panic(err)
	}
}
