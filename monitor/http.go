// Copyright © 2017 Ott-Consult UG (haftungsbeschränkt), Jörn Ott <go@ott-consult.de>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package monitor

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func (m Monitor) HttpCheck(dryrun bool, logger *log.Entry) (int64, int64, error) {
	var URL string
	var err error
	var Result Status
	var response *http.Response
	var Min time.Duration
	var Avg time.Duration
	var Max time.Duration
	var Count int64
	var LossCount int64
	var Bad int64
	var Good int64

	f := log.Fields{
		"func": "HttpCheck",
	}
	logger = logger.WithFields(f)
	logger.Debug("Start http monitor")
	Result = StatusGreen
	Min = time.Second * 86400
	for _, h := range m.Http {
		f := log.Fields{
			"Hostname": h.Hostname,
			"Https":    h.Https,
			"Port":     h.Port,
		}
		logger := logger.WithFields(f)
		if h.Https {
			URL = "https"
		} else {
			URL = "http"
		}
		URL = URL + "://" + m.IP
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{ServerName: h.Hostname},
		}
		client := &http.Client{Transport: tr}
		for _, p := range h.Path {
			logger := logger.WithField("path", p)
			u := URL + p
			req, err := http.NewRequest("GET", u, nil)
			if err != nil {
				logger.WithField("context", "Create Request").Error(err)
				Result = StatusRed
				m.err = err
				Bad++
			} else {
				if h.User != "" {
					req.SetBasicAuth(h.User, h.Password)
				}
				req.Host = h.Hostname
				//req.Header.Set("Host", h.Hostname)
				start := time.Now()
				response, err = client.Do(req)
				elapsed := time.Since(start)
				if err != nil {
					logger.WithField("context", "Send request").Error(err)
					Result = StatusRed
					m.err = err
					Bad++
				} else {
					if (response.StatusCode > 399) && (response.StatusCode != http.StatusUnauthorized) {
						logger.WithFields(log.Fields{"URL": u, "Response": response.StatusCode}).Info("Invalid HTTP response")
						Result = StatusRed
						Bad++
					} else {
						if elapsed < Min {
							Min = elapsed
						}
						if elapsed > Max {
							Max = elapsed
						}
						Avg = Avg + elapsed
						Count = Count + 1
						logger.WithFields(log.Fields{"URL": u, "Response": response.StatusCode}).Info("Good HTTP response")
						Good++
					}
				}
			}
		}
	}
	msg := fmt.Sprintf("Min: %v\nAvg: %v\nMax: %v\nSuccess: %v\nFailures: %v\n",
		strconv.FormatFloat(float64(Min.Nanoseconds())/1000, 'f', 5, 64),
		strconv.FormatFloat((float64(Avg.Nanoseconds())/1000)/float64(Count), 'f', 5, 64),
		strconv.FormatFloat(float64(Max.Nanoseconds())/1000, 'f', 5, 64),
		strconv.FormatInt(Count, 10),
		strconv.FormatInt(LossCount, 10))
	f = log.Fields{
		"Min":     Min,
		"Avg":     float64(Avg.Nanoseconds()) / float64(Count) / 1000,
		"Max":     Max,
		"Success": Count,
		"Failure": LossCount,
	}
	logger.WithFields(f).Info("Http")
	if !dryrun {
		err = m.controller.Message(Result, m.Machine, m.Http[0].Column, msg)
		if err != nil {
			m.err = err
		}
	}
	return Bad, Good, nil
}
