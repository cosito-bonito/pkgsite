// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"bytes"
	"context"
	"net/http"
	"regexp"

	"golang.org/x/discovery/internal/log"
)

const (
	latestClassPlaceholder   = "$$GODISCOVERY_LATESTCLASS$$"
	LatestVersionPlaceholder = "$$GODISCOVERY_LATESTVERSION$$"
)

// latestInfoRegexp extracts values needed to determine the latest-version badge from a page's HTML.
var latestInfoRegexp = regexp.MustCompile(`data-version="([^"]*)" data-mpath="([^"]*)" data-ppath="([^"]*)"`)

type latestFunc func(ctx context.Context, modulePath, packagePath string) string

// LatestVersion supports the badge that displays whether the version of the
// package or module being served is the latest one.
func LatestVersion(latest latestFunc) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO(b/144509703): avoid copying if possible
			crw := &capturingResponseWriter{ResponseWriter: w}
			h.ServeHTTP(crw, r)
			body := crw.bytes()
			matches := latestInfoRegexp.FindSubmatch(body)
			if matches != nil {
				version := string(matches[1])
				modulePath := string(matches[2])
				packagePath := string(matches[3])
				latestVersion := latest(r.Context(), modulePath, packagePath)
				latestClass := "DetailsHeader-"
				switch {
				case latestVersion == "":
					latestClass += "unknown"
				case latestVersion == version:
					latestClass += "latest"
				default:
					latestClass += "goToLatest"
				}
				// TODO(b/144509703): make only a single copy here, if this is slow
				body = bytes.ReplaceAll(body, []byte(latestClassPlaceholder), []byte(latestClass))
				body = bytes.ReplaceAll(body, []byte(LatestVersionPlaceholder), []byte(latestVersion))
			}
			if _, err := w.Write(body); err != nil {
				log.Errorf("LatestVersion, writing: %v", err)
			}
		})
	}
}