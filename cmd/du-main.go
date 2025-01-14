/*
 * MinIO Client (C) 2019 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"
	"net/url"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// du specific flags.
var (
	duFlags = []cli.Flag{
		cli.IntFlag{
			Name:  "depth, d",
			Usage: "print the total for a folder prefix only if it is N or fewer levels below the command line argument",
		},
	}
)

// Summarize disk usage.
var duCmd = cli.Command{
	Name:   "du",
	Usage:  "summarize disk usage folder prefixes recursively",
	Action: mainDu,
	Before: setGlobalsFromContext,
	Flags:  append(append(duFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
   MC_ENCRYPT_KEY: list of comma delimited prefix=secret values

EXAMPLES:
   1. Summarize disk usage of 'jazz-songs' bucket recursively.
      {{.Prompt}} {{.HelpName}} s3/jazz-songs

   2. Summarize disk usage of 'louis' prefix in 'jazz-songs' bucket upto two levels.
      {{.Prompt}} {{.HelpName}} --depth=2 s3/jazz-songs/louis/
`,
}

// Structured message depending on the type of console.
type duMessage struct {
	Prefix string `json:"prefix"`
	Size   string `json:"size"`
	Status string `json:"status"`
}

// Colorized message for console printing.
func (r duMessage) String() string {
	return fmt.Sprintf("%s\t%s", console.Colorize("Size", r.Size),
		console.Colorize("Prefix", r.Prefix))
}

// JSON'ified message for scripting.
func (r duMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func du(urlStr string, depth int, encKeyDB map[string][]prefixSSEPair) (int64, error) {
	targetAlias, targetURL, _ := mustExpandAlias(urlStr)
	if !strings.HasSuffix(targetURL, "/") {
		targetURL += "/"
	}

	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(urlStr), "Failed to summarize disk usage `"+urlStr+"`.")
		return 0, exitStatus(globalErrorExitStatus) // End of journey.
	}

	isRecursive := false
	isIncomplete := false
	contentCh := clnt.List(isRecursive, isIncomplete, DirFirst)
	size := int64(0)
	for content := range contentCh {
		if content.Err != nil {
			errorIf(content.Err.Trace(urlStr), "Failed to find disk usage of `"+urlStr+"` recursively.")
			return 0, exitStatus(globalErrorExitStatus)
		}

		if content.URL.String() == targetURL {
			continue
		}

		if content.Type.IsDir() {
			depth := depth
			if depth > 0 {
				depth--
			}

			subDirAlias := content.URL.Path
			if targetAlias != "" {
				subDirAlias = targetAlias + "/" + content.URL.Path
			}
			used, err := du(subDirAlias, depth, encKeyDB)
			if err != nil {
				return 0, err
			}
			size += used
		} else {
			size += content.Size
		}
	}

	if depth != 0 {
		u, err := url.Parse(targetURL)
		if err != nil {
			panic(err)
		}

		printMsg(duMessage{
			Prefix: strings.Trim(u.Path, "/"),
			Size:   strings.Join(strings.Fields(humanize.IBytes(uint64(size))), ""),
			Status: "success",
		})
	}

	return size, nil
}

// main for du command.
func mainDu(ctx *cli.Context) error {
	console.SetColor("Prefix", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// du specific flags.
	depth := ctx.Int("depth")
	if depth == 0 {
		depth = -1
	}

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	var duErr error
	for _, urlStr := range ctx.Args() {
		if _, err := du(urlStr, depth, encKeyDB); duErr == nil {
			duErr = err
		}
	}

	return duErr
}
