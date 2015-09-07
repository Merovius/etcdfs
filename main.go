// etcdfs implements a read-only fuse filesystem for accessing etcd.
//
// It accesses an etcd cluster whose address is taken from the ETCD_ENDPOINTS
// environment variable, defaulting to http://localhost:4001/. It supports
// slicing the etcd keyspace, meaning that you can mount an arbitrary
// subdirectory of etcd.
//
// Usage:
// 	etcdfs [flags…] [<subdir>] <mountpoint>
//
// Flags:
//   -allow_other
//     	Allow other users to access this filesystem
//   -allow_root
//     	Allow root to access this filesystem
//   -debug
//     	Enable debugging
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/coreos/etcd/client"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var (
	debug      = flag.Bool("debug", false, "Enable debugging")
	allowOther = flag.Bool("allow_other", false, "Allow other users to access this filesystem")
	allowRoot  = flag.Bool("allow_root", false, "Allow root to access this filesystem")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "\t%s [flags…] [<subdir>] <mountpoint>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	flag.Usage = usage
	flag.Parse()

	if *debug {
		fuse.Debug = func(v interface{}) { log.Println("[fuse]", v) }
	}

	var subdir, mountpoint string
	switch flag.NArg() {
	case 1:
		subdir = "/"
		mountpoint = flag.Arg(0)
	case 2:
		subdir = path.Join("/", flag.Arg(0))
		mountpoint = flag.Arg(1)
	default:
		usage()
		os.Exit(1)
	}

	var endpoints []string
	if ep := os.Getenv("ETCD_ENDPOINTS"); ep != "" {
		endpoints = strings.Split(ep, ",")
	} else {
		endpoints = []string{"localhost:4001"}
	}
	log.Printf("Using endpoints %v", endpoints)

	cfg := client.Config{
		Endpoints: endpoints,
	}

	etcd, err := client.New(cfg)
	if err != nil {
		return err
	}

	var mountOpts []fuse.MountOption

	if *allowOther {
		mountOpts = append(mountOpts, fuse.AllowOther())
	}
	if *allowRoot {
		mountOpts = append(mountOpts, fuse.AllowRoot())
	}
	mountOpts = append(mountOpts, fuse.DefaultPermissions())
	mountOpts = append(mountOpts, fuse.FSName("etcd:"+subdir))
	mountOpts = append(mountOpts, fuse.ReadOnly())
	mountOpts = append(mountOpts, fuse.Subtype("etcdFS"))

	log.Printf("Mounting etcd:%s to %s", subdir, mountpoint)
	c, err := fuse.Mount(
		mountpoint,
		mountOpts...,
	)
	if err != nil {
		return err
	}
	defer c.Close()

	srv := fs.New(c, nil)
	filesys := &etcdFS{
		etcd: client.NewKeysAPI(etcd),
		base: subdir,
	}

	errch := make(chan error)

	log.Printf("Start serving")
	go func() {
		errch <- srv.Serve(filesys)
	}()

	<-c.Ready
	if c.MountError != nil {
		return c.MountError
	}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	select {
	case err := <-errch:
		return err
	case s := <-sigs:
		log.Printf("Caught signal: %v", s)
		err := c.Close()
		log.Printf("Error: %v", err)
		return err
	}
}
