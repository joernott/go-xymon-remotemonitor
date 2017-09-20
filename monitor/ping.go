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
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/sparrc/go-ping"
)

func (m Monitor) PingCheck(dryrun bool, logger *log.Entry) error {
	var status Status

	f := log.Fields{
		"func":  "PingCheck",
		"Count": m.Ping.Count,
	}
	logger = logger.WithFields(f)
	logger.Debug("Start ping monitor")
	pinger, err := ping.NewPinger(m.IP)
	if err != nil {
		m.err = err
		logger.WithField("context", "new pinger").Error(err)
	} else {
		pinger.SetPrivileged(true)
		pinger.Count = m.Ping.Count
		pinger.Run()
		stats := pinger.Statistics()
		if stats.PacketLoss > 0 {
			status = StatusRed
		} else {
			status = StatusGreen
		}
		msg := fmt.Sprintf("Min: %v\nAvg: %v\nMax: %v\nLoss: %v\n",
			strconv.FormatFloat(float64(stats.MinRtt.Nanoseconds())/1000, 'f', 5, 64),
			strconv.FormatFloat(float64(stats.AvgRtt.Nanoseconds())/1000, 'f', 5, 64),
			strconv.FormatFloat(float64(stats.MaxRtt.Nanoseconds())/1000, 'f', 5, 64),
			strconv.FormatFloat(stats.PacketLoss*-1, 'f', 3, 64))
		f := log.Fields{
			"Min":  stats.MinRtt,
			"Avg":  stats.AvgRtt,
			"Max":  stats.MaxRtt,
			"Loss": stats.PacketLoss * -1,
		}
		logger.WithFields(f).Info("Ping")
		if !dryrun {
			err = m.controller.Message(status, m.Machine, m.Ping.Column, msg)
			return err
		}
	}
	return nil
}
