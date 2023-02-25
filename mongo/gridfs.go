package mongo

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"path"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

// this is to add functionality to use mongodb's gridfs - store files in mongodb

// TODO: add functions for search, put, delete

// FSlist - function to fetch the list of files together with its metadata
func (client Client) FSlist(transactionCtx context.Context, filter bson.M, sort string, order int64, skip int64,
	limit int64) ([]bson.M, error) {
	var e error

	// gridfs uses fs.files and fs.chunks as name of collection
	files, e := client.Find(transactionCtx, "fs.files", filter, bson.M{
		"sort":  sort,
		"order": order,
		"skip":  skip,
		"limit": limit,
	})
	if e != nil {
		client.log.Error("MONGO_FSLIST", e)
		return files, e
	}
	return files, e
}

// FSset - function to upload the actual file content
func (client Client) FSset(filename string) (int, error) {
	var e error

	// initialize bucket
	bucket, e := gridfs.NewBucket(client.Db)
	if e != nil {
		client.log.Error("MONGO_FSSET_NEW", e)
		return -1, e
	}

	file := path.Base(filename)
	data, e := ioutil.ReadFile(filename)
	if e != nil {
		client.log.Error("MONGO_FSSET_READ", e)
		return -1, e
	}

	uploadStream, e := bucket.OpenUploadStream(file)
	if e != nil {
		client.log.Error("MONGO_FSSET_OPEN", e)
		return -1, e
	}
	defer uploadStream.Close()

	filesize, e := uploadStream.Write(data)
	if e != nil {
		client.log.Error("MONGO_FSSET_UPLOAD", e)
		return -1, e
	}
	return filesize, e
}

// FSget - function to fetch the actual file content
func (client Client) FSget(filename string) (io.ReadSeeker, error) {
	var buf bytes.Buffer
	var f []byte
	var rs io.ReadSeeker
	var e error

	// initialize bucket
	bucket, e := gridfs.NewBucket(client.Db)
	if e != nil {
		client.log.Error("MONGO_FSGET_NEW", e)
		return rs, e
	}

	// fetch stream and store into a buffer
	_, e = bucket.DownloadToStreamByName(filename, &buf)
	if e != nil {
		client.log.Error("MONGO_FSGET_DLOAD", e)
		return nil, e
	}

	// take the whole file from the buffer for now. until native driver can support seek
	// TODO: this is dangerous
	f = buf.Bytes()

	// convert to io.ReadSeeker
	rs = bytes.NewReader(f)

	return rs, e
}
