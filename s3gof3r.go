// Package s3gof3r provides fast, parallelized, streaming access to Amazon S3. It includes a command-line interface: `gof3r`.

package s3gof3r

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Keys for an Amazon Web Services account.
// Used for signing http requests.
type Keys struct {
	AccessKey string
	SecretKey string
}

type S3 struct {
	Domain string // The s3-compatible service domain. Defaults to "s3.amazonaws.com"
	Keys
}

type Bucket struct {
	*S3
	Name string
}

type Config struct {
	*http.Client        // nil to use s3gof3r default client
	Concurrency  int    // number of parts to get or put concurrently
	PartSize     int64  //  initial  part size in bytes to use for multipart gets or puts
	NTry         int    // maximum attempts for each part
	Md5Check     bool   // the md5 hash of the object is stored in <bucket>/.md5/<object_key> and verified on gets
	Scheme       string // url scheme, defaults to 'https'
}

// Defaults
var DefaultConfig = &Config{
	Concurrency: 10,
	PartSize:    20 * mb,
	NTry:        10,
	Md5Check:    true,
	Scheme:      "https",
}

// http client timeout
const (
	clientTimeout = 5 * time.Second
)

var DefaultDomain = "s3.amazonaws.com"

// Returns a new S3
// domain defaults to DefaultDomain if empty
func New(domain string, keys Keys) *S3 {
	if domain == "" {
		domain = DefaultDomain
	}
	return &S3{domain, keys}
}

// Returns a bucket on s3j
func (s3 *S3) Bucket(name string) *Bucket {
	return &Bucket{s3, name}
}

// Provides a reader and downloads data using parallel ranged get requests.
// Data from the requests is reordered and written sequentially.
//
// Data integrity is verified via the option specified in c.
// Header data from the downloaded object is also returned, useful for reading object metadata.
func (b *Bucket) GetReader(path string, c *Config) (r io.ReadCloser, h http.Header, err error) {
	if c == nil {
		c = DefaultConfig
	}
	if c.Client == nil {
		c.Client = ClientWithTimeout(clientTimeout)
	}
	return newGetter(b.Url(path, c), c, b)
}

// Provides a writer to upload data as multipart upload requests.
//
// Each header in h is added to the HTTP request header. This is useful for specifying
// options such as server-side encryption in metadata as well as custom user metadata.
// DefaultConfig is used if c is nil.
func (b *Bucket) PutWriter(path string, h http.Header, c *Config) (w io.WriteCloser, err error) {
	if c == nil {
		c = DefaultConfig
	}
	if c.Client == nil {
		c.Client = ClientWithTimeout(clientTimeout)
	}
	return newPutter(b.Url(path, c), h, c, b)
}

// Returns a parsed url to the given path, using the scheme specified in Config.Scheme
func (b *Bucket) Url(path string, c *Config) url.URL {
	url_, err := url.Parse(fmt.Sprintf("%s://%s.%s/%s", c.Scheme, b.Name, b.S3.Domain, path))
	if err != nil {
		panic(err)
	}
	return *url_
}
