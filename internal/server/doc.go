// Package server hosts the fasthttp HTTP server, route registration,
// and middleware pipeline. It glues the api/, admin/, and embedded-UI
// handlers together with shared concerns like panic recovery, request
// logging, and CORS.
package server
