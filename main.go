package main

//go:generate go-assets-builder -s=/static static -o assets.go

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var addr, format, camera string
	flag.StringVar(&addr, "addr", ":5000", "address to serve(host:port)")
	flag.StringVar(&format, "format", "dshow", "camera format")
	flag.StringVar(&camera, "camera", "HP Truevision HD", "camera name")
	flag.Parse()

	args := []string{
		"-f", format, "-s", "320x240", "-r", "30", "-vcodec", "mjpeg", "-i",
		"video=" + camera, "-threads", "2", "-codec:v", "libx264",
		"-map", "0", "-codec:v", "libx264", "-codec:a", "libfaac",
		"-f", "segment", "-segment_format", "mpegts", "-segment_list_size", "3",
		"-segment_list_type", "m3u8", "-segment_time", "4", "-segment_list", "stream.m3u8",
		"-segment_list_flags", "+live", "stream%05d.ts",
	}

	dir, err := ioutil.TempDir("", "webcam-live-")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		time.Sleep(time.Second)
		err = os.RemoveAll(dir)
		if err != nil {
			log.Fatal(err)
		}
	}()

	cmd := exec.Command("ffmpeg", args...)
	cmd.Dir = dir
	err = cmd.Start()
	if err != nil {
		os.RemoveAll(dir)
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	log.Println(dir)
	http.Handle("/", http.FileServer(Assets))
	http.Handle("/stream/", http.StripPrefix("/stream", http.FileServer(http.Dir(dir))))

	server := &http.Server{Addr: addr, Handler: http.DefaultServeMux}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		server.Shutdown(context.Background())
	}()

	log.Printf("serving on %s", addr)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalln(err)
	}
}
