package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
)

var (
	prestart = flag.Bool("prestart", false, "run the prestart hook")
)

func exit() {
	if err := recover(); err != nil {
		if _, ok := err.(runtime.Error); ok {
			log.Println(err)
		}
		if os.Getenv("NV_DEBUG") != "" {
			log.Printf("%s", debug.Stack())
		}
		os.Exit(1)
	}
	os.Exit(0)
}

func capabilityToCLI(cap string) string {
	switch cap {
	case "compute":
		return "--compute"
	case "compat32":
		return "--compat32"
	case "graphics":
		return "--graphics"
	case "utility":
		return "--utility"
	case "video":
		return "--video"
	default:
		log.Panicln("unknown driver capability:", cap)
	}
	return ""
}

func doPrestart() {
	defer exit()
	log.SetFlags(0)

	cli := getCLIConfig()
	container := getContainerConfig()

	nvidia := container.Nvidia
	if nvidia == nil {
		// Not a GPU container, nothing to do.
		return
	}

	args := []string{cli.Path}
	if cli.LoadKmods {
		args = append(args, "--load-kmods")
	}
	if cli.Debug != nil {
		args = append(args, fmt.Sprintf("--debug=%s", *cli.Debug))
	}
	args = append(args, "configure")

	if cli.Configure.Ldconfig != nil {
		args = append(args, fmt.Sprintf("--ldconfig=%s", *cli.Configure.Ldconfig))
	}

	if len(nvidia.Devices) > 0 {
		args = append(args, fmt.Sprintf("--device=%s", nvidia.Devices))
	}

	for _, cap := range strings.Split(nvidia.Capabilities, ",") {
		if len(cap) == 0 {
			break
		}
		args = append(args, capabilityToCLI(cap))
	}

	if !cli.DisableRequire && !nvidia.DisableRequire {
		for _, req := range nvidia.Requirements {
			args = append(args, fmt.Sprintf("--require=%s", req))
		}
	}

	args = append(args, fmt.Sprintf("--pid=%s", strconv.FormatUint(uint64(container.Pid), 10)))
	args = append(args, container.Rootfs)

	log.Printf("exec command: %v", args)
	env := append(os.Environ(), cli.Environment...)
	err := syscall.Exec(cli.Path, args, env)
	log.Panicln("exec failed:", err)
}

func main() {
	flag.Parse()

	if *prestart {
		doPrestart()
	}
}
