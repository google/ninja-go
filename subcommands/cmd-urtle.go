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

import (
	"bytes"
	"context"
	"os"

	"github.com/spf13/pflag"
)

// urtleCommandSpec describes the hidden "urtle" easter-egg subcommand.
var urtleCommandSpec = &CommandSpec{
	name:   "urtle",
	desc:   "Print Urtle!",
	hidden: true,
	newCmd: func() Command {
		return new(UrtleCommand)
	},
}

// UrtleCommand executes the "urtle" subcommand.
type UrtleCommand struct {
	BaseCommand
}

// Run implements Command.
func (c *UrtleCommand) Run(ctx context.Context, fl *pflag.FlagSet) error {
	EnsureNoArgs(fl)

	// RLE encoded.
	urtle := " 13 ,3;2!2;\n8 ,;<11!;\n5 `'<10!(2`'2!\n11 ,6;, `\\. `\\9 .,c13$ec,.\n6 " +
		",2;11!>; `. ,;!2> .e8$2\".2 \"?7$e.\n <:<8!'` 2.3,.2` ,3!' ;,(?7\";2!2'<" +
		"; `?6$PF ,;,\n2 `'4!8;<!3'`2 3! ;,`'2`2'3!;4!`2.`!;2 3,2 .<!2'`).\n5 3`5" +
		"'2`9 `!2 `4!><3;5! J2$b,`!>;2!:2!`,d?b`!>\n26 `'-;,(<9!> $F3 )3.:!.2 d\"" +
		"2 ) !>\n30 7`2'<3!- \"=-='5 .2 `2-=\",!>\n25 .ze9$er2 .,cd16$bc.'\n22 .e" +
		"14$,26$.\n21 z45$c .\n20 J50$c\n20 14$P\"`?34$b\n20 14$ dbc `2\"?22$?7$c" +
		"\n20 ?18$c.6 4\"8?4\" c8$P\n9 .2,.8 \"20$c.3 ._14 J9$\n .2,2c9$bec,.2 `?" +
		"21$c.3`4%,3%,3 c8$P\"\n22$c2 2\"?21$bc2,.2` .2,c7$P2\",cb\n23$b bc,.2\"2" +
		"?14$2F2\"5?2\",J5$P\" ,zd3$\n24$ ?$3?%3 `2\"2?12$bcucd3$P3\"2 2=7$\n23$P" +
		"\" ,3;<5!>2;,. `4\"6?2\"2 ,9;, `\"?2$\n"

	count := 0
	for _, p := range []byte(urtle) {
		if '0' <= p && p <= '9' {
			count = count*10 + int(p) - '0'
		} else {
			os.Stdout.Write(bytes.Repeat([]byte{p}, max(count, 1)))
			count = 0
		}
	}
	return nil
}
