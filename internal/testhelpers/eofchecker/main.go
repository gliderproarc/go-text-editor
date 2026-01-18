package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// eofchecker writes a response without a trailing newline then exits.
func main() {
	in := bufio.NewScanner(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	if in.Scan() {
		line := strings.ToLower(strings.TrimSpace(in.Text()))
		if strings.Contains(line, "mispelt") {
			fmt.Fprint(out, "mispelt")
		}
	}
}
