# go-xymon-remotemonitor
This application is supposed to run in a cron job every 5 minutes (or whatever
interval you desire) to monitor hosts, web sites and mail servers (feature not 
yet implemented).

## Installation
Install the binary to /usr/local/bin and make sure, the file mode is 0777.

## Configuration

### Application
The application expects its configuration in /etc/xymon-client/remotemonitor.
The main configuration file is named config.json (toml or yaml also work) and
looks like this:
```
{
"server":"xymon.example.com",
"port": "1984",
"loglevel":6,
"logfile":"/var/log/remotemonitor.log",
"hostdir":"/etc/xymon-client/remotemonitor/hosts.d"
}
```
*server* is the hostname of the Xymon server, *port* is the port it is listening
on. If not defined, the default value "1984" is used.

*loglevel* is the verbosity of the logging. (1=panic, 2=fatal, 3=error, 4=warn,
5=info, 6=debug) and *logfile* is used to specify the log file name. If none is
specified, stdout is used.

The most important setting is *hostdir*. This folder contains monitor
definitions for the hosts to monitor. Each monitor corresponds to one host in
Xymon. It can contain a ping to the hosts IP, and multiple checks to http (and
https) services. Each of them can contain multiple paths. A SMTP monitor is
planned but as of 0.1.0 not yet implemented.

### Monitor
A monitor definition is saved in the *hostdir* with a file suffix of
".monitor.json" and looks like this:
```
{
  "Name":"www.example.com",
  "Machine":"www.example.com",
  "Column":"remote",
  "IP":"10.0.0.1",
  "Ping": {
    "Enabled": true,
    "Count": 5,
    "Column": "remote_ping"
  },
 "Http": [
    {
      "Https": false,
      "Hostname": "www.example.com",
      "Port": 80,
      "User": "Administrator",
      "Password": "123456",
      "Path": ["/favicon.ico", "/index.html"],
      "Column": "remote_http"
    },
    {
      "Https": true,
      "Hostname": "www.example.com",
      "Port": 443,
      "User": "Administrator",
      "Password": "123456"
      "Path": ["/favicon.ico", "/index.html", "/login.php"],
      "Column": "remote_http"
    }
  ],
  "Smtp": {
    "Enabled": false,
	"Port": 25,
	"Sender": {
		"Address": "max.muster@example.com",
		"Username": "max",
		"Password": "123456"
	},
	"Recipient": {
		"Address": "max.muster@example.com",
		"Username": "max",
		"Password": "123456"
	},
	"Subject": "Monitoring test",
	"Message": "Nothing to see here",
    "Column": "remote_smtp"
  }
}
```

The *Name* is used for logging purposes, *machine* corresponds to the host name
in Xymons hosts.cfg. The IP used to connect is specified in the *IP* field.

If you want to ping the IP, set *Ping.Enabled* to true and define the number of
pings in the *Ping.Count* field (default is 3). *Ping.Column* is the name of the
column for this check in Xymons hosts.cfg.

*Http* is an array of definitions, which allows to check multiple hostnames and
http or https in one single check. *Https* defines whether you want a http or 
https connection (default false), *Hostname* is used to set the hostname in the
connection (It is made using the IP in the URL) and *Port* defines the port to
connect to. If you need to authenticate witrh Http Basic Auth, you can supply
*Username* and *Password* for that. Leaving them empty will skip authentication.
*Path* is an array of paths on the web server (must start with a /).
Every path check must succeed for the test to be green. This allows checking for
multiple files on a web server. *Column* defines the column in Xymons hosts.cfg.

*Smtp* is currently not implemented. *Smtp.Enabled* is used to turn SMTP
checking on or off. *Smtp.Port* defines the SMTP port to connect to (may become
an array to be able to monitor multiple ports, e.g. 25, 465 and 587 in one
check).

*Smtp.Sender* and *Smtp.Recipient* are a structure containing 3 fields:
*Address* for the E-Mail address, *Username* and *Password* for authentication.

*Smtp.Subject* and *Smtp.Message* define the subject and content of the mail and
*Smtp.Column* defines the column in Xymons hosts.cfg in which the results should
turn up.

### Cronjob
To run all monitors every 5 minutes (Xymon default), create a file
/etc/cron.d/xymon-remotemonitor on the host you want the tests to run on with
the following content:
```
*/5 * * * * root /usr/local/bin/remotemonitor
```

