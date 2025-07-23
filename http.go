package wormhole

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

func (w *Wormhole) HTTP(wr http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Forwarded-Host")

	if id == "" {
		id = r.Header.Get("Host")
	}

	key := fmt.Sprintf("%s.%s", id, w.DNSManager.API.BaseDNS())

	t, ok := w.tunnels.Load(key)
	if !ok {
		http.Error(wr, "tunnel not found", http.StatusNotFound)
		return
	}

	stream, err := t.(*tunnel).session.Open()
	if err != nil {
		w.Logger.Error(fmt.Sprintf("%s: %s\n", id, ErrFailedToOpenStream))
		http.Error(wr, "failed to open stream", http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	err = w.tunnelHTTPRequest(stream, wr, r)
	if err != nil {
		errf := fmt.Sprintf("tunnel error: %s", err.Error())

		w.Logger.Error(errf)
		http.Error(wr, errf, http.StatusInternalServerError)
	}
}

func (w *Wormhole) StartHTTP() error {
	server := &http.Server{
		Addr:    w.httpAddr,
		Handler: http.HandlerFunc(w.HTTP),
	}

	go func() {
		<-w.ctx.Done()

		w.Logger.Info("context cancelled, shutting down http server")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			w.Logger.Error(fmt.Sprintf("%s: %v", fmt.Errorf("http server shutdown failed"), err))
		}
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("%v: %w", fmt.Errorf("http server stopped"), err)
	}

	return nil
}

func tunnelHTTPRequest(stream net.Conn, wr http.ResponseWriter, r *http.Request) error {
	err := r.Write(stream)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToWriteHTTPTunnelRequest, err)
	}

	bufr := bufio.NewReader(stream)

	resp, err := http.ReadResponse(bufr, r)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToReadHTTPTunnelResponse, err)
	}

	defer resp.Body.Close()

	copyHeader(wr.Header(), resp.Header)
	io.Copy(wr, resp.Body)

	return nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
