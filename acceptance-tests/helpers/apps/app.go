// Package apps manages the test app lifecycle
package apps

type App struct {
	dir       dir
	Name      string
	URL       string
	buildpack string
	memory    string
	disk      string
	manifest  string
	start     bool
}
