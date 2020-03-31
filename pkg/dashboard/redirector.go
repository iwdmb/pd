// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package dashboard

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/pingcap-incubator/tidb-dashboard/pkg/apiserver"
)

// Redirector is used to redirect when the dashboard is started in another PD.
type Redirector struct {
	mu      sync.RWMutex
	address string
	proxy   *httputil.ReverseProxy
}

// NewRedirector creates a new Redirector.
func NewRedirector() *Redirector {
	return new(Redirector)
}

// SetAddress is used to set a new address to be redirected.
func (h *Redirector) SetAddress(addr string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.address == addr {
		return
	}

	if addr == "" {
		h.address = ""
		h.proxy = nil
		return
	}

	h.address = addr
	target, _ := url.Parse(addr) // error has been handled in checkAddress
	h.proxy = httputil.NewSingleHostReverseProxy(target)
	h.proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode == http.StatusTemporaryRedirect {
			return errors.New("redirect cycle detected")
		}
		return nil
	}
}

// GetAddress is used to get the address to be redirected.
func (h *Redirector) GetAddress() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.address
}

// GetProxy is used to get the reverse proxy arriving at address.
func (h *Redirector) GetProxy() *httputil.ReverseProxy {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.proxy
}

// TemporaryRedirect sends the status code 307 to the client, and the client redirects itself.
func (h *Redirector) TemporaryRedirect(w http.ResponseWriter, r *http.Request) {
	addr := h.GetAddress()
	if addr == "" {
		apiserver.StoppedHandler.ServeHTTP(w, r)
		return
	}
	http.Redirect(w, r, addr+r.RequestURI, http.StatusTemporaryRedirect)
}

// ReverseProxy forwards the request to address and returns the response to the client.
func (h *Redirector) ReverseProxy(w http.ResponseWriter, r *http.Request) {
	proxy := h.GetProxy()
	if proxy == nil {
		apiserver.StoppedHandler.ServeHTTP(w, r)
		return
	}
	proxy.ServeHTTP(w, r)
}
