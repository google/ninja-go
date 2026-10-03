// Copyright 2025 Google LLC
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

package subcommands

// import (
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"net/url"

// 	"github.com/spf13/pflag"
// )

// const (
// 	defaultPort = 9121
// 	defaultHost = "http://localhost"
// )

// type ClientCommand struct {
// 	BaseCommand
// 	host    string
// 	port    int
// 	verbose bool
// }

// func NewClientCommand() *ClientCommand {
// 	return &ClientCommand{
// 		BaseCommand: BaseCommand{
// 			name:               "client",
// 			desc:               "Send commands to HTTP server",
// 			usage:              "",
// 			IgnoreGraph:        true,
// 			DisableInteractive: true,
// 			DisableHttp:        true,
// 		},
// 	}
// }

// func (c *ClientCommand) RegisterFlags(fl *pflag.FlagSet) {
// 	fl.StringVar(&c.host, "host", defaultHost, "Server host to send commands to")
// 	fl.IntVar(&c.port, "port", defaultPort, "Server port to send commands to")
// 	fl.BoolVarP(&c.verbose, "verbose", "v", false, "Show detail logs about the web request")
// }

// func (c *ClientCommand) Run(ctx *CommandContext, fl *pflag.FlagSet) error {
// 	return c.run(ctx, fl.Args())
// }

// func (c *ClientCommand) run(ctx *CommandContext, args []string) error {
// 	if len(args) == 0 {
// 		args = []string{"list"}
// 	}

// 	uri, err := url.Parse(fmt.Sprintf("%s:%d/cmd/%s", c.host, c.port, args[0]))
// 	if err != nil {
// 		return err
// 	}

// 	q := uri.Query()
// 	for _, c := range args[1:] {
// 		q.Add("arg", c)
// 	}
// 	uri.RawQuery = q.Encode()

// 	if c.verbose {
// 		ctx.Print("URL: %s", uri)
// 	}
// 	rsp, err := http.Get(uri.String())
// 	if err != nil {
// 		return err
// 	}
// 	defer rsp.Body.Close()
// 	body, err := io.ReadAll(rsp.Body)
// 	if err != nil {
// 		return err
// 	}
// 	ctx.Print("%s", body)
// 	return nil
// }
