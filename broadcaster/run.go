package broadcaster

import (
	"fmt"
	"net/http"
	"strconv"
)

func (b *Broadcaster) Run() error {
	b.server = &http.Server{
		Addr:    b.Address + ":" + strconv.Itoa(b.Port),
		Handler: b,
	}

	return b.server.ListenAndServe()
}

func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hey")
}
