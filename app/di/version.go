package di

// AppVersion can be set by main at startup via ldflags or direct assignment.
// It is passed to WebHandler for cache-busting static asset URLs.
var AppVersion = "dev"
