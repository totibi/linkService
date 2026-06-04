package http_handlers

import "net/http"

func RegisterSwaggerRoutes(
	mux *http.ServeMux,
) {
	mux.HandleFunc(
		"/swagger/",
		swaggerHandler,
	)

	mux.HandleFunc(
		"/swagger/swagger.json",
		swaggerJSONHandler,
	)
}

func swaggerJSONHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./generated/api.swagger.json")
}

func swaggerHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/swagger" {
		http.Redirect(
			w,
			r,
			"/swagger/?url=/swagger/swagger.json",
			http.StatusMovedPermanently,
		)
		return
	}

	http.StripPrefix(
		"/swagger/",
		http.FileServer(http.Dir("./swagger")),
	).ServeHTTP(w, r)
}
