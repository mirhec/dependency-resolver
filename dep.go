package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"

	"github.com/mholt/archiver"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	depfile  = kingpin.Flag("depfile", "The name of the dependency file. Defaults to '.dep'").Default(".dep").ExistingFile()
	registry = kingpin.Flag("registry", "The location of the registry, where to find the dependency.").URL()
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()

	usr, e := user.Current()
	if e != nil {
		log.Fatal(e)
	}

	// try to get .deprc file where the server location is specified
	deprc := usr.HomeDir + "/.deprc"
	if _, e := os.Stat(deprc); os.IsNotExist(e) {
		deprc = ".deprc"
	}
	if _, e := os.Stat(deprc); os.IsNotExist(e) {
		log.Fatal("Please create a .deprc file with all available dependency sources\n (one dependency source per line, i.e. a URL or a directory).")
	}

	sources := make([]string, 0)
	f, e := os.OpenFile(deprc, os.O_RDONLY, 0755)
	defer f.Close()
	r := bufio.NewReader(f)
	s, e := Readln(r)
	for e == nil {
		sources = append(sources, s)
		s, e = Readln(r)
	}

	// now gather all required dependencies
	dependencies := make(map[string]string)
	f, e = os.OpenFile(*depfile, os.O_RDWR, 0755)
	defer f.Close()
	r = bufio.NewReader(f)
	s, e = Readln(r)
	for e == nil {
		if !strings.HasPrefix(s, "#") {
			kv := strings.Split(s, " ")
			if len(kv) == 2 {
				dependencies[kv[0]] = kv[1]
			}
		}
		s, e = Readln(r)
	}

	// try to download the dependencies
	for dep, ver := range dependencies {
		e = nil
		for _, source := range sources {
			if _, e = Download(dep, ver, source); e == nil {
				fmt.Print("Resolved " + dep + " " + ver + "\n")
				break
			}
		}
		if e != nil {
			fmt.Printf("Error while trying to download %s: %s\n", dep, e)
		}
	}
}

// Download tries to download the dependency dep in version ver from URL url.
func Download(dep string, ver string, url string) (string, error) {
	url = url + "/" + dep + "-" + ver + ".zip"

	// Create the file
	out, err := os.Create("temp.zip")
	if err != nil {
		return url, err
	}
	defer out.Close()
	defer os.Remove("temp.zip")

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return url, err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return url, err
	}

	// Unarchive
	dir := "dep/" + dep
	err = os.RemoveAll(dir)
	if err != nil {
		return url, err
	}

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return url, err
	}

	err = archiver.Zip.Open("temp.zip", dir)
	if err != nil {
		return url, err
	}

	return url, nil
}

// Readln returns a single line (without the ending \n)
// from the input buffered reader.
// An error is returned iff there is an error with the
// buffered reader.
func Readln(r *bufio.Reader) (string, error) {
	var (
		isPrefix = true
		err      error
		line, ln []byte
	)

	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}

	return string(ln), err
}
