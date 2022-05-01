# Scriggo template engine for Fiber

Scriggo is a template engine, to see the syntax documentation please [click here](https://scriggo.com/templates)

This is based off an [article](https://code.zikani.me/using-scriggo-templates-with-the-go-fiber-web-framework) I published on my [blog](https://code.zikani.me).

```go
package main

import (
	"log"
	
	"github.com/gofiber/fiber/v2"
	fiberScriggo "github.com/zikani03/fiber-scriggo"
)

func main() {
	// Create a new engine
	engine := fiberScriggo.New("./views", ".html")

  // Or from an embedded system
  // See github.com/gofiber/embed for examples
  // engine := html.NewFileSystem(http.Dir("./views", ".html"))

	// Pass the engine to the Views
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		// Render index
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	app.Get("/layout", func(c *fiber.Ctx) error {
		// Render index within layouts/main
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main")
	})

	log.Fatal(app.Listen(":3000"))
}
```
