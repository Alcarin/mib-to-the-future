package services

import (
	"runtime"
)

type System struct{}

type InfoSistema struct {
	GoVersion string `json:"go_version"`
	GOOS      string `json:"go_os"`
	GOARCH    string `json:"go_arch"`
}

func (s *System) GetInfo() InfoSistema {
	return InfoSistema{
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
}
