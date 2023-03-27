//go:build !cgo && upgrade && ignore
// +build !cgo,upgrade,ignore

package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v50/github"
)

func download() (content []byte, err error) {
	ctx := context.Background()
	client := github.NewClientWithEnvProxy()
	release, _, err := client.Repositories.GetLatestRelease(ctx, "utelle", "SQLite3MultipleCiphers")
	if err != nil {
		return nil, err
	}

	var asset *github.ReleaseAsset
	for _, a := range release.Assets {
		if strings.HasSuffix(*a.Name, "-amalgamation.zip") {
			asset = a
		}
	}

	if asset == nil {
		return nil, fmt.Errorf("no amalgamation found in latest SQLite3MultipleCiphers release (%s)", *release.HTMLURL)
	}

	url := asset.GetBrowserDownloadURL()

	fmt.Printf("Downloading %v\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	// Ready Body Content
	content, err = io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return content, nil
}

func main() {
	fmt.Println("Go-SQLite3 Upgrade Tool")

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if filepath.Base(wd) != "upgrade" {
		log.Printf("Current directory is %q but should run in upgrade directory", wd)
		os.Exit(1)
	}

	// Download Amalgamation
	amalgamation, err := download()
	if err != nil {
		log.Fatalf("Failed to download: sqlite3mc-amalgamation; %s", err)
	}

	// Create Amalgamation Zip Reader
	rAmalgamation, err := zip.NewReader(bytes.NewReader(amalgamation), int64(len(amalgamation)))
	if err != nil {
		log.Fatal(err)
	}

	// Extract Amalgamation
	for _, zf := range rAmalgamation.File {
		var f *os.File
		ext := false
		switch path.Base(zf.Name) {
		case "sqlite3mc_amalgamation.c":
			f, err = os.Create("../sqlite3-binding.c")
		case "sqlite3mc_amalgamation.h":
			f, err = os.Create("../sqlite3-binding.h")
		case "sqlite3ext.h":
			ext = true
			f, err = os.Create("../sqlite3ext.h")
		default:
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		zr, err := zf.Open()
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.WriteString(f, "#ifndef USE_LIBSQLITE3\n")
		if err != nil {
			zr.Close()
			f.Close()
			log.Fatal(err)
		}
		if ext {
			scanner := bufio.NewScanner(zr)
			for scanner.Scan() {
				text := scanner.Text()
				if text == `#include "sqlite3.h"` {
					text = `#include "sqlite3-binding.h"
#ifdef __clang__
#define assert(condition) ((void)0)
#endif
`
				}
				_, err = fmt.Fprintln(f, text)
				if err != nil {
					break
				}
			}
			err = scanner.Err()
		} else {
			_, err = io.Copy(f, zr)
		}
		if err != nil {
			zr.Close()
			f.Close()
			log.Fatal(err)
		}
		_, err = io.WriteString(f, "#endif // !USE_LIBSQLITE3\n")
		if err != nil {
			zr.Close()
			f.Close()
			log.Fatal(err)
		}
		zr.Close()
		f.Close()
		fmt.Printf("Extracted: %v\n", filepath.Base(f.Name()))
	}

	os.Exit(0)
}
