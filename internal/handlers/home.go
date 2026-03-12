package handlers

import (
	"net/http"

	"github.com/justestif/specto/components"
)

func Home(w http.ResponseWriter, r *http.Request) {
	components.Home().Render(r.Context(), w)
}
