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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type PingMonitor struct {
	Enabled bool   `json:"Enabled" yaml:"Enabled"`
	Count   int    `json:"Count" yaml:"Count"`
	Column  string `json:"Column" yaml:"Column"`
}

type HttpMonitor struct {
	Https    bool     `json:"Https" yaml:"Https"`
	Hostname string   `json:"Hostname" yaml:"Hostname"`
	Port     int      `json:"Port" yaml:"Port"`
	Path     []string `json:"Path" yaml:"Path"`
	Column   string   `json:"Column" yaml:"Column"`
}

type MailUser struct {
	Address  string `json:"Address" yaml:"Address"`
	UserName string `json:"UserName" yaml:"UserName"`
	Password string `json:"Password" yaml:"Password"`
}

type SmtpMonitor struct {
	Enabled   bool     `json:"Enabled" yaml:"Enabled"`
	Port      int      `json:"Port" yaml:"Port"`
	Sender    MailUser `json:"Sender" yaml:"Sender"`
	Recipient MailUser `json:"Recipient" yaml:"Recipient"`
	Subject   string   `json:"Subject" yaml:"Subject"`
	Message   string   `json:"Message" yaml:"Message"`
	Column    string   `json:"Column" yaml:"Column"`
}

type Monitor struct {
	Name       string        `json:"Name" yaml:"Name"`
	Machine    string        `json:"Machine" yaml:"Machine"`
	Column     string        `json:"Column" yaml:"Column"`
	IP         string        `json:"IP" yaml: "IP"`
	Ping       PingMonitor   `json:"Ping" yaml:"Ping"`
	Http       []HttpMonitor `json:"Http" yaml:"Http"`
	Smtp       SmtpMonitor   `json:"Smtp" yaml:"Smtp"`
	controller *Controller
	err        error
}

type MonitorList map[string]Monitor

type Controller struct {
	HostDir      string
	LogLevel     int
	LogFile      string
	lf           *os.File
	XymonServer  string
	XymonPort    int
	XymonTimeout time.Duration
	monitors     MonitorList
	wait         sync.WaitGroup
}

type Status int

const StatusGreen Status = 0
const StatusYellow Status = 1
const StatusRed Status = 2

func (s Status) ToString() string {
	switch s {
	case 0:
		return "green"
	case 1:
		return "yellow"
	case 2:
		return "red"
	default:
		return "red"
	}
}

func NewController(HostDir string, XymonServer string, XymonPort int, XymonTimeout time.Duration, LogLevel int, LogFile string) (*Controller, error) {
	var c *Controller
	var err error

	c = new(Controller)
	c.HostDir = HostDir
	c.LogLevel = LogLevel
	c.LogFile = LogFile
	c.XymonServer = XymonServer
	c.XymonPort = XymonPort
	c.XymonTimeout = XymonTimeout

	err = c.configureLogging()
	if err != nil {
		return nil, err
	}
	logger := log.WithField("func", "NewController").Logger
	ok, err := dirExists(HostDir)
	if err != nil {
		logger.Fatal(err)
		return nil, err
	}
	if !ok {
		err = errors.New("Host directory '" + HostDir + "' doesn't exist.")
		logger.Fatal(err)
		return nil, err
	}
	if c.XymonServer == "" {
		err = errors.New("XymonServer must not be empty.")
		logger.Fatal(err)
		return nil, err
	}
	err = c.loadMonitors()
	if err != nil {
		return nil, err
	}
	if len(c.monitors) == 0 {
		err = errors.New("No monitors defined.")
		logger.Warn(err)
	} else {
		logger.WithField("monitors", len(c.monitors)).Debug("Monitors defined")
	}

	return c, nil
}

func (c *Controller) configureLogging() error {
	var err error

	if c.LogFile == "" {
		log.SetOutput(os.Stdout)
	} else {
		c.lf, err = os.OpenFile(c.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		log.SetOutput(c.lf)
	}
	switch c.LogLevel {
	case 6:
		log.SetLevel(log.DebugLevel)
	case 5:
		log.SetLevel(log.InfoLevel)
	case 4:
		log.SetLevel(log.WarnLevel)
	case 3:
		log.SetLevel(log.ErrorLevel)
	case 2:
		log.SetLevel(log.FatalLevel)
	case 1:
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.WarnLevel)
		log.Warn("Illegal Debug level, defaulting to Info (5)")
	}
	return nil
}

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (c *Controller) loadMonitors() error {
	var err error
	var filename string
	logger := log.WithField("func", "loadMonitors")
	files, err := ioutil.ReadDir(c.HostDir)
	if err != nil {
		logger.WithFields(log.Fields{"context": "List monitor definitions", "file": c.HostDir}).Fatal(err)
		return err
	}
	c.monitors = make(MonitorList)
	for _, f := range files {
		filename = f.Name()
		if strings.HasSuffix(filename, ".monitor.json") {
			m, err := c.readJsonMonitor(filename)
			if err == nil {
				m.controller = c
				c.monitors[filename] = m
			}
		}
		/* ToDo: Support yaml and support lists */
	}
	return nil
}

func (c *Controller) readJsonMonitor(filename string) (Monitor, error) {
	var err error
	var m Monitor

	logger := log.WithField("func", "readJsonMonitor")
	rawJson, err := ioutil.ReadFile(c.HostDir + string(os.PathSeparator) + filename)
	if err != nil {
		logger.WithFields(log.Fields{"context": "Read monitor definition", "file": filename}).Error(err)
		return m, err
	}
	err = json.Unmarshal(rawJson, &m)
	if err != nil {
		logger.WithFields(log.Fields{"context": "Parse monitor definition", "file": filename}).Error(err)
		return m, err
	}
	return m, nil
}

func (c *Controller) Run(dryrun bool) error {
	logger := log.WithField("func", "Controller.Run")
	logger.Debug("Running monitors")
	for n, m := range c.monitors {
		c.wait.Add(1)
		l := logger.WithField("monitor", n)
		l.Debug("Running")
		go m.Run(dryrun, l)
	}
	c.wait.Wait()
	return nil
}

func (c *Controller) Message(Color Status, Machine string, Column string, Message string) error {
	var err error

	logger := log.WithField("func", "Message").Logger

	conn, err := net.DialTimeout("tcp", c.XymonServer+":"+strconv.Itoa(c.XymonPort), c.XymonTimeout)
	if err != nil {
		logger.WithField("context", "Connect to XYMon").Error(err)
		return err
	}
	defer conn.Close()
	t := time.Now().Format(time.RFC3339)
	Output := fmt.Sprintf("status 	%v.%v %v %v\n\n%v", Machine, Column, Color.ToString(), t, Message)
	f := log.Fields{
		"Machine":     Machine,
		"Column":      Column,
		"Color":       Color,
		"ColorString": Color.ToString(),
		"Timestamp":   t,
	}
	logger.WithFields(f).Debug(Message)
	_, err = conn.Write([]byte(Output))
	if err != nil {
		logger.WithField("context", "Send to XYMon").Error(err)
		return err
	}
	// Xymon waiting that write connection has been closed to send response...
	conn.(*net.TCPConn).CloseWrite()

	s, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		logger.WithField("context", "XYMon response").Error(err)
		return err
	}
	logger.WithField("context", "XYMon response").Debug(s)
	return nil
}

func (m Monitor) Run(dryrun bool, logger *log.Entry) {
	var Good int64
	var Bad int64

	defer m.controller.wait.Done()
	f := log.Fields{
		"func":    "monitor.Run",
		"dryrun":  dryrun,
		"machine": m.Machine,
		"column":  m.Column,
		"ip":      m.IP,
	}
	logger = logger.WithFields(f)
	logger.Debug("Start monitor")
	if m.Ping.Enabled {
		err := m.PingCheck(dryrun, logger)
		if err != nil {
			m.err = err
			Bad++
		} else {
			Good++
		}
	}
	if len(m.Http) > 0 {
		b, g, err := m.HttpCheck(dryrun, logger)
		if err != nil {
			m.err = err
		}
		Bad = Bad + b
		Good = Good + g
	}
	if m.Smtp.Enabled {
		err := m.SmtpCheck(dryrun, logger)
		if err != nil {
			m.err = err
			Bad++
		} else {
			Good++
		}
	}
	msg := fmt.Sprintf("Bad: %v\nGood: %v\n",
		strconv.FormatInt(Bad, 10),
		strconv.FormatInt(Good, 10))
	f = log.Fields{
		"Bad":  Bad,
		"Good": Good,
	}
	logger.WithFields(f).Info("Monitor")
	if !dryrun {
		var status Status
		if Bad == 0 {
			status = StatusGreen
		} else {
			if Good == 0 {
				status = StatusRed
			} else {
				status = StatusYellow
			}
		}
		m.controller.Message(status, m.Machine, m.Column, msg)
	}
}
