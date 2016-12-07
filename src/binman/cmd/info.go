//
// Copyright © 2016 Ikey Doherty <ikey@solus-project.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package cmd

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"libeopkg"
	"os"
	"strings"
)

var infoCmd = &cobra.Command{
	Use:   "info [file.eopkg]",
	Short: "inspect a package",
	Long: `Emit information for a binary .eopkg file to the console.
This is to provide a bridge for those without access to eopkg.`,
	Example: "binman info nano-*.eopkg",
	RunE:    infoPackage,
}

func init() {
	RootCmd.AddCommand(infoCmd)
}

// infoPackage will examine the specified package and emit information
// for it, akin to "eopkg info" output.
func infoPackage(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("You must supply a filename")
	}

	pkg, err := libeopkg.Open(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open for reading: %v\n", err)
		return nil
	}
	defer pkg.Close()

	if err := pkg.ReadMetadata(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read package: %v\n", err)
		return nil
	}

	metaPkg := pkg.Meta.Package
	upd := metaPkg.History[0]
	table := tablewriter.NewWriter(os.Stdout)
	var deps []string
	for _, dep := range metaPkg.RuntimeDependencies {
		deps = append(deps, dep.Name)
	}
	output := [][]string{
		[]string{"Package file", args[0]},
		[]string{"Name", fmt.Sprintf("%s, version: %s, release: %d\n", metaPkg.Name, upd.Version, upd.Release)},
		[]string{"Summary", metaPkg.Summary},
		[]string{"Description", metaPkg.Description},
		[]string{"Licenses", strings.Join(metaPkg.License, " ")},
		[]string{"Component", metaPkg.PartOf},
		[]string{"Distribution", fmt.Sprintf("%s, Dist. Release: %s\n", metaPkg.Distribution, metaPkg.DistributionRelease)},
		[]string{"Dependencies", strings.Join(deps, " ")},
	}
	table.SetBorder(false)
	table.SetColumnSeparator(":")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(output)
	table.Render()

	return nil
}
