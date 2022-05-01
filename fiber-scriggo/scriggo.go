package scriggo

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/utils"
	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

// Engine struct
type Engine struct {
	// views folder
	directory string
	// http.FileSystem supports embedded files
	fileSystem http.FileSystem
	// views extension
	extension string
	// layout variable name that incapsulates the template
	layout string
	// determines if the engine parsed all templates
	loaded bool
	// reload on each render
	reload bool
	// debug prints the parsed templates
	debug bool
	// lock for funcmap and templates
	mutex sync.RWMutex
	// template funcmap
	funcmap native.Declarations
	// templates
	templates map[string]string
	// scriggo filesystem
	fsys scriggo.Files
}

// New returns a Scriggo render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		directory: directory,
		extension: extension,
		layout:    "embed",
		funcmap:   make(native.Declarations),
	}
	engine.AddFunc(engine.layout, func() error {
		return fmt.Errorf("layout called unexpectedly.")
	})
	return engine
}

func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{
		directory:  "/",
		fileSystem: fs,
		extension:  extension,
		layout:     "embed",
		funcmap:    make(native.Declarations),
	}
	engine.AddFunc(engine.layout, func() error {
		return fmt.Errorf("layout called unexpectedly.")
	})
	return engine
}

// Layout defines the variable name that will incapsulate the template
func (e *Engine) Layout(key string) *Engine {
	e.layout = key
	return e
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (e *Engine) Delims(left, right string) *Engine {
	fmt.Println("delims: this method is not supported for scriggo")
	return e
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn native.Declaration) *Engine {
	e.mutex.Lock()
	e.funcmap[name] = fn
	e.mutex.Unlock()
	return e
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you don't want to restart
// the application when you edit a template file.
func (e *Engine) Reload(enabled bool) *Engine {
	e.reload = enabled
	return e
}

// Debug will print the parsed templates when Load is triggered.
func (e *Engine) Debug(enabled bool) *Engine {
	e.debug = enabled
	return e
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.templates = make(map[string]string)
	e.fsys = scriggo.Files{}

	// add scriggo builtins
	// e.funcmap = native.Declarations{
	// 	"capitalize": builtin.Capitalize, // global function
	// }

	// Loop trough each directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}
		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}
		// Skip file if it does not equal the given template extension
		if len(e.extension) >= len(path) || path[len(path)-len(e.extension):] != e.extension {
			return nil
		}
		// Get the relative file path
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.directory, path)
		if err != nil {
			return err
		}
		// Reverse slashes '\' -> '/' and
		// partials\footer.tmpl -> partials/footer.tmpl
		name := filepath.ToSlash(rel)
		// Remove ext from name 'index.tmpl' -> 'index'
		name = strings.TrimSuffix(name, e.extension)
		// name = strings.Replace(name, e.extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.fileSystem)
		if err != nil {
			return err
		}

		e.fsys[filepath.ToSlash(rel)] = buf
		e.templates[name] = filepath.ToSlash(rel)

		// Debugging
		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	// notify engine that we parsed all templates
	e.loaded = true
	if e.fileSystem != nil {
		return utils.Walk(e.fileSystem, e.directory, walkFn)
	}
	err := filepath.Walk(e.directory, walkFn)
	// for key := range e.fsys {
	// 	fmt.Println("Loaded template", key)
	// }
	return err
}

// Render will execute the template name along with the given values.
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	if !e.loaded || e.reload {
		if e.reload {
			e.loaded = false
		}
		if err := e.Load(); err != nil {
			return err
		}
	}
	templatePath, ok := e.templates[template]
	if !ok {
		return fmt.Errorf("render: template %s does not exist", template)
	}

	opts := &scriggo.BuildOptions{
		Globals: e.funcmap,
	}

	// Register the variables as Globals for Scriggo's type checking to work
	for key, value := range binding.(fiber.Map) {
		opts.Globals[key] = value
	}

	// Build the template.
	tmpl, err := scriggo.BuildTemplate(e.fsys, templatePath, opts)
	if err != nil {
		return err
	}

	return tmpl.Run(out, nil, nil)
}
