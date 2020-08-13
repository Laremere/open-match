// Copyright 2019 Google LLC
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

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	{ // Check running directory.
		_, err := os.Stat("tool.go")
		if err != nil {
			fmt.Fprint(os.Stderr, "Unable to detect tool.go, please run from the root Open Match directory.\n")
			os.Exit(1)
		}
	}

	var commands map[string]command
	commands = map[string]command{
		"help": command{
			f: func([]string) {
				for k, v := range commands {
					fmt.Println(k, ":", v.helpText)
				}
			},
			helpText: "Prints this help text.",
		},
		"library": command{
			f: func([]string) {
				fmt.Println("Run `go run tool.go libexplain <name> to see the details for a command.")
				for _, book := range library {
					fmt.Printf(" %s (%s) - %s\n", book.name, book.tool.cmd, book.helpText)
				}
			},
			helpText: "A list of handy commands for developing or using Open Match.",
		},
		"libexplain": command{
			f: func(args []string) {
				var book *book
				for _, b := range library {
					// TODO: ensure value passed.
					if b.name == args[0] {
						book = b
						break
					}
				}
				if book == nil {
					panic("whoops")
				}
				fmt.Printf("To run this command now, run `go run tool.go librun %s\n", book.name)
				fmt.Println()
				fmt.Print(book.tool.cmd)
				for _, arg := range book.args {
					fmt.Print(" ", arg.value)
				}
				fmt.Println()
				fmt.Println()
				fmt.Println("Step by step explanation:")
				fmt.Println(" ", book.tool.cmd, "->", book.tool.helpText)
				for _, arg := range book.args {
					fmt.Println(" ", arg.value, "->", arg.helpText)
				}
			},
			helpText: "Show a command from the library with an explanation.",
		},
	}

	if len(os.Args) < 2 {
		fmt.Println("Command required.  Run `go run tool.go help` for more info.")
		os.Exit(1)
	}
	c, ok := commands[os.Args[1]]
	if !ok {
		fmt.Println("Unknown command \"", os.Args[1], "\".  Run `go run tool.go help` for more info.")
		os.Exit(1)
	}
	c.f(os.Args[2:])

	// runCommand("git", "status")

	// path, err := os.Getwd()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(path)
	// fmt.Println(os.Args[1:])
	// err = checkKubectlVersion()

	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Hello build.")

	// w := &resetWriter{w: os.Stdout}

	// _, err := fmt.Fprint(w, "thinking1234")
	// if err != nil {
	// 	panic(err)
	// }
	// time.Sleep(time.Second)
	// err = w.reset()
	// if err != nil {
	// 	panic(err)
	// }
	// _, err = fmt.Fprintln(w, "thought")
	// if err != nil {
	// 	panic(err)
	// }
}

type command struct {
	f        func([]string)
	helpText string
}

// func checkKubectlVersion() error {
// 	o, err := runCommand("kubectl", "version", "--output=json")
// 	if err != nil {
// 		return err
// 	}

// 	// o, err := c.Output()
// 	// if err != nil {
// 	// 	return err
// 	// }

// 	// fmt.Println(string(o))
// 	return nil
// }

func runCommand(cmd string, args ...string) ([]byte, error) {
	fmt.Println("Running command:")
	fmt.Print("\t", cmd)
	for _, arg := range args {
		fmt.Print(" ", arg)
	}
	fmt.Println()
	c := exec.Command(cmd, args...)
	o, err := c.Output()
	if err != nil {
		// fmt.Print("Attempted to run this command:\n\t", cmd)
		fmt.Print("\nHowever it encountered this error\n\t", err, "\n")
	}
	return o, nil
}

// type resetWriter struct {
// 	w io.Writer
// 	n int
// }

// func (r *resetWriter) Write(b []byte) (int, error) {
// 	n, err := r.w.Write(b)
// 	r.n += n
// 	return n, err
// }

// var backspaces []byte

// func init() {
// 	backspaces = make([]byte, 1024)
// 	for i := range backspaces {
// 		backspaces[i] = 8 // ASCII backspace
// 	}
// }

// func (r *resetWriter) reset() error {
// 	for r.n > 0 {
// 		rem := len(backspaces)
// 		if rem > r.n {
// 			rem = r.n
// 		}

// 		n, err := r.w.Write(backspaces[:rem])
// 		// fmt.Println(r.n, n, rem)
// 		r.n -= n
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

type book struct { // static command?
	name     string
	helpText string
	tool     *tool
	args     []arg
}

type arg struct {
	value    string
	helpText string
}

type tool struct {
	cmd        string
	helpText   string
	installUrl string
}

var gcloud = &tool{
	cmd:        "gcloud",
	helpText:   "gcloud is the primary command-line interface tool to create and manage Google Cloud resources.",
	installUrl: "https://cloud.google.com/sdk/install",
}

var library = []*book{
	{
		"create-gke-cluster",
		"Creates a GKE cluster for running Open Match in.",
		gcloud,
		[]arg{
			{
				"container",
				"deploy and manage clusters of mahcines for running containers",
			},
			{
				"clusters",
				"deploy and teardown Google Kubernetes Engine clusters",
			},
			{
				"create",
				"create a cluster",
			},
			{
				"om-cluster",
				"the name of the cluster being created",
			},
			{
				"--machine-type=n1-standard-4",
				"The type of machine to use for nodes. 4 cores.",
			},
			{
				"--enable-autoscaling",
				"Enables autoscaling for a node pool.",
			},
			{
				"--min-nodes=1",
				"Minimum number of nodes in the node pool.",
			},
			{
				"--num-nodes=2",
				"",
			},
			{
				"--max-nodes=10",
				"Maximum number of nodes in the node pool.",
			},
			{
				"--disk-size=50",
				"",
			},
			{
				"--no-enable-basic-auth",
				"",
			},
			{
				"--no-issue-client-certificate",
				"",
			},
			{
				"--enable-ip-alias",
				"",
			},
			{
				"--metadata disable-legacy-endpoints=true",
				"",
			},
			{
				"--enable-autoupgrade",
				"",
			},
			{
				"--enable-pod-security-policy",
				"",
			},
			{
				"--image-type=cos_containerd",
				"",
			},
			{
				"--tags open-match",
				"",
			},
		},
	},
}
