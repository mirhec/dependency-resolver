package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
	"github.com/spf13/viper"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	depfile  = kingpin.Flag("depfile", "The name of the dependency file. Defaults to '.dep'").Default(".dep").ExistingFile()
	registry = kingpin.Flag("registry", "The location of the registry, where to find the dependency.").URL()
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()

	viper.SetDefault("SevenZipExecutable", "C:/Program Files/7-Zip/7z.exe")
	viper.SetDefault("DependencyDirectory", "dep")
	viper.SetDefault("Repositories", []string{})
	viper.SetConfigName("config")       // name of config file (without extension)
	viper.AddConfigPath("$HOME/.deprc") // call multiple times to add many search paths
	viper.AddConfigPath(".")            // optionally look for config in the working directory
	viper.AutomaticEnv()
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	// now gather all required dependencies
	dependencies := make(map[string]string)
	f, e := os.OpenFile(*depfile, os.O_RDWR, 0755)
	defer f.Close()
	r := bufio.NewReader(f)
	s, e := Readln(r)
	for e == nil {
		if !strings.HasPrefix(s, "#") {
			kv := strings.Split(s, " ")
			if len(kv) == 2 {
				dependencies[kv[0]] = kv[1]
			}
		}
		s, e = Readln(r)
	}

	err = os.RemoveAll(viper.GetString("DependencyDirectory"))
	if err != nil {
		panic(err)
	}
	err = os.Mkdir(viper.GetString("DependencyDirectory"), 0755)
	if err != nil {
		panic(err)
	}

	// try to download the dependencies
	for dep, ver := range dependencies {
		e = nil
		for _, source := range viper.GetStringSlice("Repositories") {
			if _, e = Download(dep, ver, source, ".zip"); e == nil {
				fmt.Print("Resolved " + dep + " " + ver + "\n")
				break
			} else if _, e = Download(dep, ver, source, ".7z"); e == nil {
				fmt.Print("Resolved " + dep + " " + ver + "\n")
				break
			} else if _, e = CopyFromDisk(dep, ver, source); e == nil {
				fmt.Print("Resolved " + dep + " " + ver + "\n")
				break
			}
		}
		if e != nil {
			fmt.Printf("Error while trying to download %s %s: %s\n", dep, ver, e)
		}
	}
}

// CopyFromDisk tries to copy the dependency dep in version ver from folder dir
func CopyFromDisk(dep string, ver string, dir string) (string, error) {
	path := dir + "/" + dep + "-" + ver + ".*"
	files, _ := filepath.Glob(path)

	if len(files) == 0 {
		return path, fmt.Errorf("CopyFromDisk: dependency could not be found")
	}
	path = files[0]

	if strings.HasSuffix(path, ".zip") || strings.HasSuffix(path, ".7z") {
		file, err := ioutil.TempFile(".", "temp_")
		if err != nil {
			return path, err
		}
		defer os.Remove(file.Name())
		defer file.Close()

		err = CopyFile(path, file)
		if err != nil {
			return path, err
		}

		dest := viper.GetString("DependencyDirectory") + "/" + dep
		if strings.HasSuffix(path, ".zip") {
			err = archiver.Zip.Open(file.Name(), dest)
			if err != nil {
				return path, err
			}
		} else {
			file.Close()
			cmd := exec.Command("7z", "e", file.Name(), "-o"+dest, "-r", "-t7z", "-aoa")
			_, err := cmd.CombinedOutput()
			if err != nil {
				return path, err
			}
		}
	} else {
		file, err := os.Create(viper.GetString("DependencyDirectory") + "/" + dep + filepath.Ext(path))
		if err != nil {
			return path, err
		}
		defer file.Close()

		err = CopyFile(path, file)
		if err != nil {
			return path, err
		}
	}

	return path, nil
}

// CopyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func CopyFile(src string, out *os.File) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// Download tries to download the dependency dep in version ver from URL url.
func Download(dep string, ver string, url string, ext string) (string, error) {
	url = url + "/" + dep + "-" + ver + ext

	// Create the file
	// out, err := os.Create("temp" + ext)
	file, err := ioutil.TempFile(".", "temp_")
	if err != nil {
		return url, err
	}
	defer os.Remove(file.Name())
	defer file.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return url, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return url, fmt.Errorf("Download: The file could not be downloaded")
	}

	// Write the body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return url, err
	}

	// Unarchive
	dir := viper.GetString("DependencyDirectory") + "/" + dep
	// err = os.Remove(dir)
	// if err != nil {
	// 	return url, err
	// }

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return url, err
	}

	if ext == ".zip" {
		err = archiver.Zip.Open(file.Name(), dir)
		if err != nil {
			return url, err
		}
	} else if ext == ".7z" {
		file.Close()
		cmd := exec.Command("7z", "e", file.Name(), "-o"+dir, "-r", "-t7z", "-aoa")
		_, err := cmd.CombinedOutput()
		if err != nil {
			return url, err
		}
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
