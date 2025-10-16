package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	if b[0] == '"' {
		sd := string(b[1 : len(b)-1])
		dur, err := time.ParseDuration(sd)
		if err != nil {
			return err
		}
		*d = Duration(dur)
		return nil
	}

	var id int64
	id, err = json.Number(string(b)).Int64()
	*d = Duration(id)
	return err
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return fmt.Appendf(nil, `"%s"`, time.Duration(d).String()), nil
}

type ServerConfig struct {
	ListenAddr   string   `json:"listen_addr"`
	ReadTimeout  Duration `json:"read_timeout"`
	WriteTimeout Duration `json:"write_timeout"`
	IdleTimeout  Duration `json:"idle_timeout"`
}
