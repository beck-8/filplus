package main

var CurrentCommit string

// BuildVersion is the local build version
const BuildVersion = "0.1.0"

func UserVersion() string {
	return BuildVersion + "+git." + CurrentCommit
}
