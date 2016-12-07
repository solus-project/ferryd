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
	"github.com/spf13/cobra"
	"libeopkg"
	"os"
	"strings"
	"text/tabwriter"
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
	if err := pkg.ReadMetadata(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read package: %v\n", err)
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 1, 8, 2, '\t', 0)

	metaPkg := pkg.Meta.Package
	upd := metaPkg.History[0]
	fmt.Fprintf(writer, "Package file\t: %s\n", args[0])
	fmt.Fprintf(writer, "Name\t: %s, version: %s, release: %d\n", metaPkg.Name, upd.Version, upd.Release)
	fmt.Fprintf(writer, "Summary\t: %s\n", metaPkg.Summary)
	fmt.Fprintf(writer, "Description\t: %s", metaPkg.Description)
	fmt.Fprintf(writer, "Licenses\t: %s\n", strings.Join(metaPkg.License, " "))
	fmt.Fprintf(writer, "Component\t: %s\n", metaPkg.PartOf)
	fmt.Fprintf(writer, "Distribution\t: %s, Dist. Release: %s\n", metaPkg.Distribution, metaPkg.DistributionRelease)
	var deps []string
	for _, dep := range metaPkg.RuntimeDependencies {
		deps = append(deps, dep.Name)
	}
	fmt.Fprintf(writer, "Dependencies\t: %s\n", strings.Join(deps, " "))
	writer.Flush()
	defer pkg.Close()
	return nil
}
