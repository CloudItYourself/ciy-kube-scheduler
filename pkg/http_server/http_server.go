package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
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
	ret, err := ciyHttp.ciySortPlugin.Score(ciyHttp.ctx, nil, nil, vars["nodeName"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Error getting metrics for: %w\n", err)
		w.Write([]byte("0"))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(string(ret)))
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
	tlsCertPath := os.Getenv("SCHEDULER_CERT_PATH")
	tlsKeyath := os.Getenv("SCHEDULER_KEY_PATH")

	certProvider := CertReloader{CertFile: tlsCertPath,
		KeyFile:           tlsKeyath,
		CachedCert:        nil,
		CachedCertModTime: time.Now(),
		CachedKeyModTime:  time.Now()}

	var httpListener net.Listener
	tlsConfig := &tls.Config{
		NextProtos:     []string{"http/1.1"},
		GetCertificate: certProvider.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}

	var err error
	httpServer.TLSConfig = tlsConfig
	httpListener, err = tls.Listen("tcp", addr, tlsConfig)

	if err == nil {
		httpServer.Serve(httpListener)
	}
	fmt.Errorf("failed to bind to TCP address: %w", err)
 os.Exit(1)
}
