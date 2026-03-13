package uci

import (
	"fmt"
	"strings"
)

// Option represents a UCI engine option.
type Option struct {
	Name    string
	Type    string // "spin", "check", "string", "button", "combo"
	Default interface{}
	Min     int
	Max     int
	Value   interface{}
}

// Options holds all engine options.
type Options struct {
	Hash         int
	MoveOverhead int // milliseconds reserved for communication overhead
	Threads      int
	Ponder       bool
	SyzygyPath   string
	ShowWDL      bool
	UseNNUE      bool
	EvalFile     string
}

func DefaultOptions() Options {
	return Options{
		Hash:         64,
		MoveOverhead: 10,
		Threads:      1,
		UseNNUE:      true,
	}
}

// PrintOptions outputs UCI option declarations.
func (o *Options) PrintOptions(printf func(format string, a ...interface{})) {
	printf("option name Hash type spin default %d min 1 max 4096\n", o.Hash)
	printf("option name Threads type spin default %d min 1 max 128\n", o.Threads)
	printf("option name Ponder type check default false\n")
	printf("option name Move Overhead type spin default %d min 0 max 5000\n", o.MoveOverhead)
	printf("option name SyzygyPath type string default %s\n", o.SyzygyPath)
	printf("option name UCI_ShowWDL type check default false\n")
	printf("option name UseNNUE type check default true\n")
	evalDefault := o.EvalFile
	if evalDefault == "" {
		evalDefault = "<embedded>"
	}
	printf("option name EvalFile type string default %s\n", evalDefault)
}

// SetOption applies a UCI setoption command.
func (o *Options) SetOption(name, value string) error {
	switch strings.ToLower(name) {
	case "hash":
		var v int
		if _, err := fmt.Sscanf(value, "%d", &v); err != nil {
			return err
		}
		if v < 1 || v > 4096 {
			return fmt.Errorf("Hash value out of range: %d", v)
		}
		o.Hash = v
	case "threads":
		var v int
		if _, err := fmt.Sscanf(value, "%d", &v); err != nil {
			return err
		}
		if v < 1 || v > 128 {
			return fmt.Errorf("Threads value out of range: %d", v)
		}
		o.Threads = v
	case "ponder":
		o.Ponder = strings.ToLower(value) == "true"
	case "syzygypath":
		o.SyzygyPath = value
	case "uci_showwdl":
		o.ShowWDL = strings.ToLower(value) == "true"
	case "move overhead":
		var v int
		if _, err := fmt.Sscanf(value, "%d", &v); err != nil {
			return err
		}
		if v < 0 || v > 5000 {
			return fmt.Errorf("Move Overhead value out of range: %d", v)
		}
		o.MoveOverhead = v
	case "usennue":
		o.UseNNUE = strings.ToLower(value) == "true"
	case "evalfile":
		o.EvalFile = value
	default:
		// Ignore unknown options per UCI spec.
	}
	return nil
}
