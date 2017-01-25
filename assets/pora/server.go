package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"path/filepath"
)

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/env", env)
	http.HandleFunc("/write", write)
	http.HandleFunc("/create", createFile)
	http.HandleFunc("/read/", readFile)
	http.HandleFunc("/delete/", deleteFile)
	fmt.Println("listening...")

	ports := os.Getenv("PORT")
	portArray := strings.Split(ports, " ")

	errCh := make(chan error)

	for _, port := range portArray {
		println(port)
		go func(port string) {
			errCh <- http.ListenAndServe(":"+port, nil)
		}(port)
	}

	err := <-errCh
	if err != nil {
		panic(err)
	}
}

type VCAPApplication struct {
	InstanceIndex int `json:"instance_index"`
}

func hello(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "instance index: %s", os.Getenv("INSTANCE_INDEX"))
}

func getPath() string {
	r, err := regexp.Compile("\"container_(dir|path)\": \"([^\"]+)\"")
	if err != nil {
		panic(err)
	}

	vcapEnv := os.Getenv("VCAP_SERVICES")
	match := r.FindStringSubmatch(vcapEnv)
	if len(match) < 3 {
		fmt.Fprintf(os.Stderr, "VCAP_SERVICES is %s", vcapEnv)
		panic("failed to find container_dir in environment json")
	}

	return match[2]
}

func write(res http.ResponseWriter, req *http.Request) {
	mountPointPath := getPath() + "/poratest-" + randomString(10)

	d1 := []byte("Hello Persistent World!\n")
	err := ioutil.WriteFile(mountPointPath, d1, 0644)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Writing \n"))
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	body, err := ioutil.ReadFile(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Reading \n"))
		res.Write([]byte(err.Error()))
		return
	}

	err = os.Remove(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Deleting \n"))
		res.Write([]byte(err.Error()))
		return
	}

	res.Write(body)
	return
}

func createFile(res http.ResponseWriter, req *http.Request) {
	fileName := "pora" + randomString(10)
	mountPointPath := filepath.Join(getPath(), fileName)

	d1 := []byte("Hello Persistent World!\n")
	err := ioutil.WriteFile(mountPointPath, d1, 0644)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fileName))
	return
}

func readFile(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	fileName := parts[len(parts) - 1]
	mountPointPath := filepath.Join(getPath(), fileName)

	body, err := ioutil.ReadFile(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write(body)
	res.Write([]byte("instance index: " + os.Getenv("INSTANCE_INDEX")))
	return
}

func deleteFile(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	fileName := parts[len(parts) - 1]
	mountPointPath := filepath.Join(getPath(), fileName)

	err := os.Remove(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte("deleted " + fileName))
	return
}

func env(res http.ResponseWriter, req *http.Request) {
	for _, e := range os.Environ() {
		fmt.Fprintf(res, "%s\n", e)
	}
}

func randomString(n int) string {
	runes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}
