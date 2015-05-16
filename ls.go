/*
 * Minio Client (C) 2015 Minio, Inc.
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

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

/// LS - related internal functions

// iso8601 date
const (
	printDate = "2006-01-02 15:04:05 MST"
)

// printContent prints content meta-data
func printContent(date time.Time, v int64, name string, fileType os.FileMode) {
	fmt.Printf(console.Time("[%s] ", date.Local().Format(printDate)))
	fmt.Printf(console.Size("%6s ", humanize.IBytes(uint64(v))))

	// just making it explicit
	switch fileType.IsDir() {
	case true:
		// if one finds a prior suffix no need to append a new one
		if strings.HasSuffix(name, "/") {
			fmt.Println(console.Dir("%s", name))
		} else {
			fmt.Println(console.Dir("%s/", name))
		}
	default:
		fmt.Println(console.File("%s", name))
	}
}

// doList - list all entities inside a folder
func doList(clnt client.Client, targetURL string) error {
	var err error
	for contentCh := range clnt.List() {
		if contentCh.Err != nil {
			err = contentCh.Err
			break
		}
		printContent(contentCh.Content.Time, contentCh.Content.Size, contentCh.Content.Name, contentCh.Content.Type)
	}
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	return nil
}

// doListRecursive - list all entities inside folders and sub-folders recursively
func doListRecursive(clnt client.Client, targetURL string) error {
	var err error
	for contentCh := range clnt.ListRecursive() {
		if contentCh.Err != nil {
			err = contentCh.Err
			break
		}
		// this special handling is necessary since we are sending back absolute paths with in ListRecursive()
		// a user would not wish to see a URL just for recursive and not for regular List()
		//
		// To be consistent we have to filter them out
		contentName := strings.TrimPrefix(contentCh.Content.Name, strings.TrimSuffix(targetURL, "/")+"/")
		printContent(contentCh.Content.Time, contentCh.Content.Size, contentName, contentCh.Content.Type)
	}
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	return nil
}