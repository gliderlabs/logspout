/*

func syslogStreamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","
	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		tag := logline.Name + target.AppendTag
		remote, err := syslog.Dial("udp", target.Addr, syslog.LOG_USER|syslog.LOG_INFO, tag)
		assert(err, "syslog")
		io.WriteString(remote, logline.Data)
	}
}


func rfc5424Streamer(target Target, types []string, logstream chan *Log) {
	typestr := "," + strings.Join(types, ",") + ","

	pri := syslog.LOG_USER | syslog.LOG_INFO
	hostname, _ := os.Hostname()

	c, err := net.Dial("udp", target.Addr)
	assert(err, "net dial rfc5424")

	if hostname == "" {
		hostname = c.LocalAddr().String()
	}

	for logline := range logstream {
		if typestr != ",," && !strings.Contains(typestr, logline.Type) {
			continue
		}
		tag := logline.Name + target.AppendTag
		nl := ""
		if !strings.HasSuffix(logline.Data, "\n") {
			nl = "\n"
		}

		timestamp := time.Now().Format(time.RFC3339)
		_, err := fmt.Fprintf(c, "<%d>1 %s %s %s %d - [%s] %s%s", pri, timestamp, hostname, tag, os.Getpid(), target.StructuredData, logline.Data, nl)
		assert(err, "rfc5424")
	}
}
*/
