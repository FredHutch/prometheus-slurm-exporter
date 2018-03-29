/* Copyright 2017 Victor Penso, Matteo Dessalvi

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>. */

package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"net/http"
	"os"
)

func init() {
	// Metrics have to be registered to be exposed
	//prometheus.MustRegister(NewSchedulerCollector()) // from scheduler.go
	//prometheus.MustRegister(NewQueueCollector())     // from queue.go
	prometheus.MustRegister(NewNodesCollector()) // from nodes.go
}

var listenAddress = flag.String("listen-address", "", "The address to listen on for HTTP requests.")
var cluster = flag.String("cluster", "", "Required: Slurm cluster name (gizmo or bealge)")

func main() {
	flag.Parse()

	// the port flag is required
	if *listenAddress == "" {
		fmt.Fprintf(os.Stderr, "\nMissing required --listenAddress flag\n\n")
		flag.Usage()
		os.Exit(2)
	}

	// the cluster flag is required
	if *cluster == "" {
		fmt.Fprintf(os.Stderr, "\nMissing required --cluster flag\n\n")
		flag.Usage()
		os.Exit(2)
	}

	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	log.Infof("Starting Server: %s", ":"+*listenAddress)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+*listenAddress, nil))
}
