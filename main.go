package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/mattn/go-isatty"
	"github.com/mkideal/cli"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

type argT struct {
	cli.Helper
	File   string `cli:"f,file" usage:"input file with nodes"`
	Script string `cli:"s,script" usage:"text file containing list of commands to be executed in the remote bash"`
	LogDir string `cli:"log-dir" usage:"save all operations and output per nodes in the directory. If empty string, the logs not created" dft:"./logs"`
	NoLogs bool   `cli:"no-logs" usage:"don't create the host logs'"`
}

type Host struct {
	Hostname string
	IP       string
}

type Result struct {
	name   string
	ip     string
	output string
	err    error
}

func (argv *argT) Validate(ctx *cli.Context) error {
	if isatty.IsTerminal(os.Stdin.Fd()) {
		//No data is being piped to stdin
		if argv.File == "" {
			return fmt.Errorf("I need a node list:\n<node-name>  <node-IP>  <any other data>")
		}
	} else {
		//Data is being piped to stdin
		if argv.File != "" {
			return fmt.Errorf("ambiguous input data - stdin and file %s", argv.File)
		}
	}

	return nil
}

func main() {
	cli.Run(new(argT), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)

		command := strings.Join(ctx.Args(), " ")
		return run(argv.File, argv.Script, command, argv)
	})
}

func run(nodeFile, scriptFile, command string, argv *argT) error {
	if scriptFile != "" && command != "" {
		return fmt.Errorf("ambiguous command - in arguments and script file %s", scriptFile)
	}
	if scriptFile == "" && command == "" {
		return fmt.Errorf("neither command nor script is not specified")
	}

	var tty *os.File
	var err error

	var ttyReader *bufio.Reader

	tty, err = os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer tty.Close()

	ttyReader = bufio.NewReader(tty)

	if !argv.NoLogs {
		if _, err = os.Stat(argv.LogDir); os.IsNotExist(err) {
			err = os.MkdirAll(argv.LogDir, 0750)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	var script []byte

	if scriptFile != "" {
		script, err = os.ReadFile(scriptFile)
		if err != nil {
			return fmt.Errorf("can't read script file %s: %v", script, err)
		}
	}

	if nodeFile == "-" || nodeFile == "" {
		nodeFile = "/dev/stdin"
	}

	file, err := os.Open(nodeFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var wg sync.WaitGroup
	resultsChannel := make(chan Result)

	var hosts []Host

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) >= 2 {
			name := fields[0]
			ip := fields[1]
			hosts = append(hosts, Host{
				Hostname: name,
				IP:       ip,
			})
		}
	}

	switch {
	case len(hosts) == 0:
		fmt.Printf("No hosts in the input!\n")
		os.Exit(2)
	case script != nil:
		fmt.Printf("The script %s will be run on\n%d hosts:\n%s\nAre you sure [y/N]?", script, len(hosts), hostsStr(hosts))
	case command != "":
		fmt.Printf("The command\n==========\n%s\n==========\nwill be run on %d hosts:\n%s\nAre you sure [y/N]?", command, len(hosts), hostsStr(hosts))
	}
	response, err := ttyReader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}
	response = strings.ToLower(strings.TrimSpace(response))
	switch response {
	case "y", "yes":
	case "n", "no", "":
		return fmt.Errorf("user chose to abort")
	default:
		return fmt.Errorf("invalid input. Please enter y or n")
	}

	logDir := argv.LogDir
	if argv.NoLogs {
		logDir = ""
	}

	for _, host := range hosts {
		wg.Add(1)
		switch {
		case script != nil:
			go runScript(host, script, &wg, resultsChannel, logDir)
		case command != "":
			go runCommand(host, command, &wg, resultsChannel, logDir)
		}
	}

	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	for result := range resultsChannel {
		if result.err != nil {
			fmt.Printf(">>> %s/%s:\n------------------------------\nERROR: %s------------------------------\n\n", result.name, result.ip, result.output)
		} else {
			fmt.Printf(">>> %s/%s:\n------------------------------\n%s------------------------------\n\n", result.name, result.ip, result.output)
		}
	}

	return nil
}

func hostsStr(hosts []Host) string {
	var builder strings.Builder
	for i, host := range hosts {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(host.Hostname)
	}
	return builder.String()
}

func runCommand(host Host, command string, wg *sync.WaitGroup, resultsChannel chan Result, logDir string) {
	defer wg.Done()

	var logFile *os.File
	var err error

	if logDir != "" {
		logFileName := path.Join(logDir, host.Hostname+".log")
		logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Can't open log file %s\n", logFileName)
			os.Exit(2)
		}
		defer logFile.Close()

		logFile.WriteString("=== " + time.Now().Format(time.UnixDate) + " COMMAND ===============\n")
		logFile.WriteString(command)
		logFile.WriteString("\n")
		logFile.WriteString("--------------------------------------------------------\n")
	}

	cmd := exec.Command("ssh", "-o", "BatchMode=yes", host.IP, command)

	output, err := cmd.CombinedOutput()

	resultsChannel <- Result{
		name:   host.Hostname,
		ip:     host.IP,
		output: string(output),
		err:    err,
	}

	if logDir != "" {
		logFile.Write(output)
		logFile.WriteString("\n")

		logFile.WriteString("--------------------------------------------------------\n")
	}

}

func runScript(host Host, script []byte, wg *sync.WaitGroup, resultsChannel chan Result, logDir string) {
	defer wg.Done()

	var logFile *os.File
	var err error

	if logDir != "" {
		logFileName := path.Join(logDir, host.Hostname+".log")
		logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Can't open log file %s\n", logFileName)
			os.Exit(2)
		}
		defer logFile.Close()

		logFile.WriteString("=== " + time.Now().Format(time.UnixDate) + " SCRIPT ================\n")
		logFile.Write(script)
		logFile.WriteString("\n")
		logFile.WriteString("--------------------------------------------------------\n")
	}

	cmd := exec.Command("ssh", "-o", "BatchMode=yes", host.IP, "bash -s")
	cmd.Stdin = bytes.NewReader(script)

	output, err := cmd.CombinedOutput()

	resultsChannel <- Result{
		name:   host.Hostname,
		ip:     host.IP,
		output: string(output),
		err:    err,
	}

	if logDir != "" {
		logFile.Write(output)
		logFile.WriteString("\n")

		logFile.WriteString("--------------------------------------------------------\n")
	}
}
