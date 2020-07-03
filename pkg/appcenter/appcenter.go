package appcenter

import (
	"context"
	"fmt"
	"strings"
)

const (
	separator = "/"
)

var (
	registry = make(map[string]InstallFunc)
)

// Config is the configuration that is devoted to an application.
type Config map[string]interface{}

// Settings contains the configurations of an application and all its sub-applications.
type Settings interface {
	// Config returns the corresponding configuration of the current application.
	Config() Config

	// Apps returns the configurations of all sub-applications.
	Apps() map[string]Settings

	// Installed returns a list of sub-application names that will be installed
	// within the current application.
	Installed() []string
}

// NewFunc creates an application. It will return an error if failed.
type NewFunc func(ctx context.Context, config Config) (*App, error)

// MountFunc mounts the sub-applications specified by subApps.
//
// For example, when developing HTTP applications, we can use MountFunc to mount
// the HTTP routes of sub-applications on the HTTP route of their parent application.
type MountFunc func(ctx context.Context, subApps []*App) error

// CleanFunc does the cleanup work for the application.
type CleanFunc func() error

// InstallFunc installs an application and all its sub-applications recursively.
// It will return an error if failed.
type InstallFunc func(ctx context.Context, settings Settings) (*App, error)

// App represents an application.
type App struct {
	Name      string
	MountFunc MountFunc
	CleanFunc CleanFunc

	// The use-case specific options, which are customized by users.
	Options interface{}

	subApps []*App
}

// SubApps returns all the sub-applications of the current application.
func (a *App) SubApps() []*App {
	return a.subApps
}

// Uninstall does the cleanup work for the application and all its sub-applications
// recursively. It will return the first error it encountered, if there are more
// than one error.
func (a *App) Uninstall() error {
	for _, subApp := range a.subApps {
		if err := subApp.Uninstall(); err != nil {
			return err
		}
	}

	if a.CleanFunc != nil {
		return a.CleanFunc()
	}

	return nil
}

// Register registers an application with the given registration name and the
// installation function.
func Register(name string, newApp NewFunc) error {
	name = strings.TrimSuffix(name, separator)

	if _, ok := registry[name]; ok {
		return fmt.Errorf("app %q already exists", name)
	}
	registry[name] = makeInstallFunc(name, newApp)
	return nil
}

// Unregister unregisters the given applications specified by names. It will
// clear the registry (i.e. unregister all applications) if no name is provided.
func Unregister(names ...string) {
	if len(names) == 0 {
		// Clear the registry.
		registry = make(map[string]InstallFunc)
	}

	for _, name := range names {
		delete(registry, name)
	}
}

func makeRegistrationName(parent, name string) string {
	return parent + separator + name
}

func makeInstallFunc(registrationName string, newApp NewFunc) InstallFunc {
	return func(ctx context.Context, settings Settings) (*App, error) {
		app, err := newApp(ctx, settings.Config())
		if err != nil {
			return nil, err
		}

		for _, name := range settings.Installed() {
			subRegistrationName := makeRegistrationName(registrationName, name)
			install, ok := registry[subRegistrationName]
			if !ok {
				return nil, fmt.Errorf("no app registered with name %q", subRegistrationName)
			}

			subSettings, ok := settings.Apps()[name]
			if !ok {
				return nil, fmt.Errorf("settings of app %q is not found", name)
			}

			subApp, err := install(ctx, subSettings)
			if err != nil {
				return nil, err
			}

			app.subApps = append(app.subApps, subApp)
		}

		if app.MountFunc != nil {
			app.MountFunc(ctx, app.subApps)
		}

		return app, nil
	}
}

func InstallRoot(ctx context.Context, settings Settings, appName string, newApp NewFunc) (*App, error) {
	install := makeInstallFunc(appName, newApp)
	return install(ctx, settings)
}
