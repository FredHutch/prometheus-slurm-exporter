/* Copyright 2020 Joeri Hermans, Victor Penso, Matteo Dessalvi

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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"os/exec"
	"strings"
  "regexp"
	"strconv"
)

type GPUsMetrics struct {
	alloc       float64
	idle        float64
	total       float64
	utilization float64
}

func GPUsGetMetrics() *GPUsMetrics {
	return ParseGPUsMetrics()
}

func ReturnGPUCountFromGres(gres string) float64 {
  var node_gpus = 0.0
  for _, v := range strings.Split(gres, ","){
    gpu_gres_line, _ := regexp.Match(`gpu:*`, []byte(v))
    if gpu_gres_line {
      gpu_gres_data := strings.Split(v, ":")
      /* GRES descriptor can be of the form "gpu:<count>" or
        "gpu:<type>:<count>". If there are more than 2 elements in
        the array we can assume there's a "type" defined.

        "no_consume" isn't supported here */
      if len(gpu_gres_data) > 2 {
        node_gpus, _ = strconv.ParseFloat(gpu_gres_data[2], 64)
      } else {
        node_gpus, _ = strconv.ParseFloat(gpu_gres_data[1], 64)
      }
      break
    }
  }
  return node_gpus
}

func ParseAllocTRES(tresline string) float64 {
  /* return allocated gres/gpu from output of squeue */
  /* billing=5,cpu=5,gres/gpu=1,node=1 */
  var num_gpus = 0.0
  for _, v := range strings.Split(tresline, ","){
    gpu_entry, _ := regexp.Match("gres/gpu=*", []byte(v))
    if gpu_entry {
      count := strings.Split(v, "=")
      num_gpus, _ = strconv.ParseFloat(count[1], 64)
      break
    }
  }
  return num_gpus
}

func ParseAllocatedGPUs() float64 {
	var num_gpus = 0.0

  args := []string{"-O", "tres-alloc:", "-r", "--noheader"}
	output := string(Execute("squeue", args))
	if len(output) > 0 {
		for _, line := range strings.Split(output, "\n") {
			if len(line) > 0 {
				line = strings.Trim(line, "\"")
        /*
				descriptor := strings.TrimPrefix(line, "gpu:")
				job_gpus, _ := strconv.ParseFloat(descriptor, 64)
        */
        job_gpus := ParseAllocTRES(line)
				num_gpus += job_gpus
			}
		}
	}

	return num_gpus
}

func ParseTotalGPUs() float64 {
	var num_gpus = 0.0
	var node_gpus = 0.0

	args := []string{"-h", "-o \"%n %G\""}
	output := string(Execute("sinfo", args))
	if len(output) > 0 {
		for _, line := range strings.Split(output, "\n") {
			if len(line) > 0 {
				line = strings.Trim(line, "\"")
				descriptor := strings.Fields(line)[1]
        /* gres line may have multiple entries, find the one starting
           with "gpu:"
        */
        for _, v := range strings.Split(descriptor, ","){
          gpu_gres_line, _ := regexp.Match(`gpu:*`, []byte(v))
          if gpu_gres_line {
            gpu_gres_data := strings.Split(v, ":")
            /* GRES descriptor can be of the form "gpu:<count>" or
              "gpu:<type>:<count>". If there are more than 2 elements in
              the array we can assume there's a "type" defined.

              "no_consume" isn't supported here */
            if len(gpu_gres_data) > 2 {
              node_gpus, _ = strconv.ParseFloat(gpu_gres_data[2], 64)
            } else {
              node_gpus, _ = strconv.ParseFloat(gpu_gres_data[1], 64)
            }
            num_gpus += node_gpus
            break
          }
        }
			}
		}
	}

	return num_gpus
}

func ParseGPUsMetrics() *GPUsMetrics {
	var gm GPUsMetrics
	total_gpus := ParseTotalGPUs()
	allocated_gpus := ParseAllocatedGPUs()
	gm.alloc = allocated_gpus
	gm.idle = total_gpus - allocated_gpus
	gm.total = total_gpus
	gm.utilization = allocated_gpus / total_gpus
	return &gm
}

// Execute the sinfo command and return its output
func Execute(command string, arguments []string) []byte {
	cmd := exec.Command(command, arguments...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	out, _ := ioutil.ReadAll(stdout)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return out
}

/*
 * Implement the Prometheus Collector interface and feed the
 * Slurm scheduler metrics into it.
 * https://godoc.org/github.com/prometheus/client_golang/prometheus#Collector
 */

func NewGPUsCollector() *GPUsCollector {
	return &GPUsCollector{
		alloc: prometheus.NewDesc("slurm_gpus_alloc", "Allocated GPUs", nil, nil),
		idle:  prometheus.NewDesc("slurm_gpus_idle", "Idle GPUs", nil, nil),
		total: prometheus.NewDesc("slurm_gpus_total", "Total GPUs", nil, nil),
		utilization: prometheus.NewDesc("slurm_gpus_utilization", "Total GPU utilization", nil, nil),
	}
}

type GPUsCollector struct {
	alloc       *prometheus.Desc
	idle        *prometheus.Desc
	total       *prometheus.Desc
	utilization *prometheus.Desc
}

// Send all metric descriptions
func (cc *GPUsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cc.alloc
	ch <- cc.idle
	ch <- cc.total
	ch <- cc.utilization
}
func (cc *GPUsCollector) Collect(ch chan<- prometheus.Metric) {
	cm := GPUsGetMetrics()
	ch <- prometheus.MustNewConstMetric(cc.alloc, prometheus.GaugeValue, cm.alloc)
	ch <- prometheus.MustNewConstMetric(cc.idle, prometheus.GaugeValue, cm.idle)
	ch <- prometheus.MustNewConstMetric(cc.total, prometheus.GaugeValue, cm.total)
	ch <- prometheus.MustNewConstMetric(cc.utilization, prometheus.GaugeValue, cm.utilization)
}
