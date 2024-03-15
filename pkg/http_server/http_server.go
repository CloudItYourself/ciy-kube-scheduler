package httpserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	mux "github.com/gorilla/mux"
	ciySort "sigs.k8s.io/scheduler-plugins/pkg/ciy_sort_plugin"
)

const (
	HTTPReadTimeout = 30 * time.Second
)

type CiyHttpServer struct {
	ciySortPlugin *ciySort.CiySortPlugin
	ctx           context.Context
}

func (ciyHttp *CiyHttpServer) getNodeScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	isNodePersistent, err := ciyHttp.ciySortPlugin.IsNodePersistent(vars["nodeName"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("50"))
	}
	ret, frameworkError := ciyHttp.ciySortPlugin.GetCiyScore(ciyHttp.ctx, vars["nodeName"], isNodePersistent)
	if frameworkError != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Error getting metrics for: %w\n", frameworkError.AsError())
		w.Write([]byte("0"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(string(int64(ret * 100))))
	}
}

func RunHttpServer() {
	ciyHttpServer := CiyHttpServer{ciySort.NewCiySortPlugin(), context.Background()}
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/node_score/{nodeName}", ciyHttpServer.getNodeScore)
	addr := os.Getenv("SCHEDULER_SCORE_SERVER_URL")

	httpServer := &http.Server{
		Addr:        addr,
		Handler:     router,
		ReadTimeout: HTTPReadTimeout,
		// Go does not handle timeouts in HTTP very well, and there is
		// no good way to handle streaming timeouts, therefore we need to
		// keep this at unlimited and be careful to clean up connections
		// https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/#aboutstreaming
		WriteTimeout: 0,
	}

	var httpListener net.Listener

	var err error
	httpListener, err = net.Listen("tcp", addr)

	if err == nil {
		httpServer.Serve(httpListener)
		log.Fatalf("Httpserver stopped serving")
	} else {
		log.Fatalf("failed to bind to TCP address:: %s", err.Error())
	}
}
