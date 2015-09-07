package main

import (
	"hash/crc64"
	"log"
	"os"
	"path"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

var (
	// crcTable is a precalculated crc64 table for inode number computations.
	crcTable = crc64.MakeTable(crc64.ECMA)
)

// inode calculates the inode number of a given key as the CRC64 checksum.
func inode(key string) uint64 {
	return crc64.Checksum([]byte(key), crcTable)
}

// etcdFS implements fs.FS.
type etcdFS struct {
	etcd client.KeysAPI
	base string
}

// node returns an fs.Node corresponding to the given key. It distinguishes
// between files and directories and correctly translates etcd errors into
// appropriate syscall errors.
func (f *etcdFS) node(ctx context.Context, key string) (fs.Node, error) {
	resp, err := f.etcd.Get(ctx, key, &client.GetOptions{
		Sort:   true,
		Quorum: true,
	})
	if err != nil {
		log.Printf("Error fetching node %q: %v", key, err)
		if _, ok := err.(client.Error); !ok {
			return nil, err
		}

		e := err.(client.Error)
		switch e.Code {
		case client.ErrorCodeKeyNotFound:
			return nil, fuse.Errno(syscall.ENOENT)
		case client.ErrorCodeNotDir:
			return nil, fuse.Errno(syscall.ENOTDIR)
		case client.ErrorCodeUnauthorized:
			return nil, fuse.Errno(syscall.EPERM)
		default:
			return nil, err
		}
	}

	n := resp.Node
	if n.Dir {
		return &etcdDir{f, n}, nil
	}
	return &etcdFile{f, n}, nil
}

// Root implements fs.FS.
func (f *etcdFS) Root() (fs.Node, error) {
	return f.node(context.Background(), f.base)
}

// etcdDir implements fs.Node for a directory node.
type etcdDir struct {
	fs *etcdFS
	n  *client.Node
}

// Attr implements fs.Node.
func (d *etcdDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = inode(d.n.Key)
	attr.Mode = os.ModeDir | 0555
	//	attr.Uid = uint32(os.Getuid())
	//	attr.Gid = uint32(os.Getgid())
	return nil
}

// ReadDirAll implements fs.HandleReadDirAller.
func (d *etcdDir) ReadDirAll(ctx context.Context) (ret []fuse.Dirent, err error) {
	for _, n := range d.n.Nodes {
		var typ fuse.DirentType
		if n.Dir {
			typ = fuse.DT_Dir
		} else {
			typ = fuse.DT_File
		}

		ret = append(ret, fuse.Dirent{
			Inode: inode(n.Key),
			Name:  path.Base(n.Key),
			Type:  typ,
		})
	}
	return ret, nil
}

// Lookup ipmlements fs.NodeStringLookuper.
func (d *etcdDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	return d.fs.node(ctx, path.Join(d.n.Key, name))
}

// etcdFile implements fs.Node for a leaf node.
type etcdFile struct {
	fs *etcdFS
	n  *client.Node
}

// Attr implements fs.Node.
func (f *etcdFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = inode(f.n.Key)
	attr.Size = uint64(len(f.n.Value))
	attr.Mode = 0444
	//	attr.Uid = uint32(os.Getuid())
	//	attr.Gid = uint32(os.Getgid())
	return nil
}

// ReadAll implements HandleReadAller.
func (f *etcdFile) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(f.n.Value), nil
}
