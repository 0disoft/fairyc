package main

import "fmt"

type hostSmokeMode uint8

const (
	hostSmokeCorrectness hostSmokeMode = iota
	hostSmokeWithMeasurements
)

func parseHostSmokeMode(arguments []string) (hostSmokeMode, error) {
	if len(arguments) == 0 {
		return hostSmokeCorrectness, nil
	}
	if len(arguments) == 1 && arguments[0] == "--with-measurements" {
		return hostSmokeWithMeasurements, nil
	}
	return hostSmokeCorrectness, fmt.Errorf(
		"usage: fairyc-windows-host-smoke [--with-measurements]",
	)
}
