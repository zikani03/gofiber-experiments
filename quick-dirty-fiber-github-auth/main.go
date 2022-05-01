package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	fiberScriggo "github.com/zikani03/fiber-scriggo"
)

func staticPage(templateFile, title string) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.Render(templateFile, fiber.Map{"Title": title})
	}
}

func main() {
	githubProvider := github.New(os.Getenv("GITHUB_KEY"), os.Getenv("GITHUB_SECRET"), os.Getenv("AUTH_CALLBACK_URL"))

	goth.UseProviders(githubProvider)

	gothic.Store = NewProviderStore()

	engine := fiberScriggo.New("./templates", ".html")

	engine.Debug(true)

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login", http.StatusPermanentRedirect)
	})

	app.Get("/login", staticPage("login", "Login"))
	app.Get("/auth/error", staticPage("auth_error", "Auth Error"))

	gothLoginMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), "provider", "github")
			reqCtx := req.WithContext(ctx)
			gothUser, err := gothic.CompleteUserAuth(res, reqCtx)
			if err == nil {
				ctx := context.WithValue(req.Context(), "user", gothUser)
				fmt.Println("authenticated ", gothUser)
				//store.Storage.Set("user", gothUser, time.Duration(24*time.Hour.Hours()))

				next.ServeHTTP(res, req.WithContext(ctx))
				return
			} else {
				gothic.BeginAuthHandler(res, reqCtx)
			}
		})
	}

	gothCallbackMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			gothUser, err := gothic.CompleteUserAuth(res, req)
			if err == nil {
				ctx := context.WithValue(req.Context(), "user", gothUser)
				fmt.Println("authenticated ", gothUser)
				next.ServeHTTP(res, req.WithContext(ctx))
				return
			}

			next.ServeHTTP(res, req)
		})
	}

	app.Get("/auth/github", adaptor.HTTPMiddleware(gothLoginMiddleware))

	app.Get("/auth/github/callback", adaptor.HTTPMiddleware(gothCallbackMiddleware), func(c *fiber.Ctx) error {
		gothUser := goth.User{} // TODO; Find a way to get this user from the damn session
		c.Locals("user", gothUser)
		return c.Redirect("/home")
	})

	app.Get("/home", adaptor.HTTPMiddleware(gothCallbackMiddleware), func(c *fiber.Ctx) error {
		user := c.Locals("user")
		fmt.Println("Found the user: ", user)
		return c.Render("userinfo", fiber.Map{
			"Title": "User Info",
			"user":  &user,
		})
	})

	app.Get("/auth/logout", func(c *fiber.Ctx) error {
		// gothic.Logout(adaptor.FiberHandler(c))
		return c.Redirect("/", http.StatusTemporaryRedirect)
	})

	err := app.Listen(":3000")
	if err != nil {
		panic(err)
	}
}
